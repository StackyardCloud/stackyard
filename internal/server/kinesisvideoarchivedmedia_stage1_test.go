package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type kvsListedFragment struct {
	FragmentNumber               string
	FragmentSizeInBytes          int64
	ProducerTimestamp            float64
	ServerTimestamp              float64
	FragmentLengthInMilliseconds int64
}

func kvsSeedArchive(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	ts := newKVSMediaTestServer(t)
	arn := kvsCreateStream(t, ts, "archive", 24)
	if status, _ := kvsPutMedia(t, ts, "archive", nil, kvsTestMKV(0, 2000, 4000, 6000)); status != http.StatusOK {
		t.Fatalf("seed PutMedia: %d", status)
	}
	return ts, arn
}

func TestKinesisVideoArchivedMediaStage1Fragments(t *testing.T) {
	ts, arn := kvsSeedArchive(t)

	var page struct {
		Fragments []kvsListedFragment
		NextToken string
	}
	resp := kvsMediaJSON(t, ts, "/listFragments", `{"StreamARN":"`+arn+`","MaxResults":3}`)
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &page); err != nil || len(page.Fragments) != 3 || page.NextToken == "" {
		t.Fatalf("first page: %v %+v", err, page)
	}
	if f := page.Fragments[1]; f.ProducerTimestamp != 1700000002 || f.FragmentLengthInMilliseconds != 33 || f.FragmentSizeInBytes == 0 {
		t.Fatalf("fragment fields: %+v", f)
	}
	all := append([]kvsListedFragment{}, page.Fragments...)
	resp = kvsMediaJSON(t, ts, "/listFragments", `{"StreamARN":"`+arn+`","MaxResults":3,"NextToken":"`+page.NextToken+`"}`)
	page.NextToken = ""
	if err := json.Unmarshal(mustBody(t, resp), &page); err != nil || len(page.Fragments) != 1 || page.NextToken != "" {
		t.Fatalf("second page: %v %+v", err, page)
	}
	all = append(all, page.Fragments...)

	resp = kvsMediaJSON(t, ts, "/listFragments", `{"StreamARN":"`+arn+`","FragmentSelector":{"FragmentSelectorType":"PRODUCER_TIMESTAMP","TimestampRange":{"StartTimestamp":1700000001,"EndTimestamp":1700000004}}}`)
	if err := json.Unmarshal(mustBody(t, resp), &page); err != nil || len(page.Fragments) != 2 {
		t.Fatalf("ranged list: %v %+v", err, page)
	}
	for name, body := range map[string]string{
		"bad token":  `{"StreamARN":"` + arn + `","NextToken":"Zm9v"}`,
		"bad max":    `{"StreamARN":"` + arn + `","MaxResults":0}`,
		"half range": `{"StreamARN":"` + arn + `","FragmentSelector":{"FragmentSelectorType":"SERVER_TIMESTAMP","TimestampRange":{"StartTimestamp":1}}}`,
	} {
		resp = kvsMediaJSON(t, ts, "/listFragments", body)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s: %d", name, resp.StatusCode)
		}
		resp.Body.Close()
	}

	resp = kvsMediaJSON(t, ts, "/getMediaForFragmentList", `{"StreamName":"archive","Fragments":["`+all[3].FragmentNumber+`","`+all[0].FragmentNumber+`"]}`)
	if got := kvsFragmentNumbersIn(t, mustBody(t, resp)); len(got) != 2 || got[0] != all[3].FragmentNumber {
		t.Fatalf("GetMediaForFragmentList order: %v", got)
	}
	resp = kvsMediaJSON(t, ts, "/getMediaForFragmentList", `{"StreamName":"archive","Fragments":["123"]}`)
	assertStatus(t, resp, http.StatusNotFound)
	resp = kvsMediaJSON(t, ts, "/getMediaForFragmentList", `{"StreamName":"archive","Fragments":[]}`)
	assertStatus(t, resp, http.StatusBadRequest)

	clip := func(start, end string) *http.Response {
		return kvsMediaJSON(t, ts, "/getClip", `{"StreamName":"archive","ClipFragmentSelector":{"FragmentSelectorType":"PRODUCER_TIMESTAMP","TimestampRange":{"StartTimestamp":`+start+`,"EndTimestamp":`+end+`}}}`)
	}
	resp = clip("1700000002", "1700000006")
	if got := kvsFragmentNumbersIn(t, mustBody(t, resp)); resp.Header.Get("Content-Type") != "video/webm" || len(got) != 3 {
		t.Fatalf("GetClip: %s %v", resp.Header.Get("Content-Type"), got)
	}
	assertStatus(t, clip("1600000000", "1600000010"), http.StatusNotFound)
	assertStatus(t, clip("1700000006", "1700000002"), http.StatusBadRequest)

	kvsCreateStream(t, ts, "no-archive", 0)
	resp = kvsMediaJSON(t, ts, "/getClip", `{"StreamName":"no-archive","ClipFragmentSelector":{"FragmentSelectorType":"SERVER_TIMESTAMP","TimestampRange":{"StartTimestamp":1,"EndTimestamp":2}}}`)
	if body := mustBody(t, resp); resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "NoDataRetentionException") {
		t.Fatalf("GetClip without retention: %d %s", resp.StatusCode, body)
	}
}

func TestKinesisVideoArchivedMediaStage1Images(t *testing.T) {
	ts, _ := kvsSeedArchive(t)
	var out struct {
		Images []struct {
			TimeStamp    float64
			Error        string
			ImageContent string
		}
		NextToken string
	}
	body := `{"StreamName":"archive","ImageSelectorType":"PRODUCER_TIMESTAMP","StartTimestamp":1699999998,"EndTimestamp":1700000008,"SamplingInterval":2000,"Format":"PNG","WidthPixels":32,"MaxResults":4}`
	resp := kvsMediaJSON(t, ts, "/getImages", body)
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil || len(out.Images) != 4 || out.NextToken == "" {
		t.Fatalf("GetImages page: %v %+v", err, out)
	}
	if out.Images[0].Error != "NO_MEDIA" || out.Images[1].ImageContent == "" {
		t.Fatalf("coverage: %+v", out.Images[:2])
	}
	raw, _ := base64.StdEncoding.DecodeString(out.Images[1].ImageContent)
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil || img.Bounds().Dx() != 32 || img.Bounds().Dy() != 18 {
		t.Fatalf("placeholder image: %v %v", err, img)
	}
	resp = kvsMediaJSON(t, ts, "/getImages", strings.Replace(body, `"MaxResults":4`, `"MaxResults":4,"NextToken":"`+out.NextToken+`"`, 1))
	out.NextToken = ""
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil || len(out.Images) != 2 || out.NextToken != "" {
		t.Fatalf("GetImages second page: %v %+v", err, out)
	}
	for _, bad := range []string{
		strings.Replace(body, `"PNG"`, `"GIF"`, 1),
		strings.Replace(body, `"Format":"PNG"`, `"Format":"PNG","FormatConfig":{"JPEGQuality":"80"}`, 1),
		strings.Replace(body, `"WidthPixels":32`, `"WidthPixels":4000`, 1),
	} {
		assertStatus(t, kvsMediaJSON(t, ts, "/getImages", bad), http.StatusBadRequest)
	}
}

func kvsGetURL(t *testing.T, rawURL string) (int, string, string) {
	t.Helper()
	resp, err := http.Get(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header.Get("Content-Type"), string(b)
}

func TestKinesisVideoArchivedMediaStage1Playback(t *testing.T) {
	ts, _ := kvsSeedArchive(t)

	resp := kvsMediaJSON(t, ts, "/getHLSStreamingSessionURL", `{"StreamName":"archive","PlaybackMode":"ON_DEMAND","HLSFragmentSelector":{"FragmentSelectorType":"PRODUCER_TIMESTAMP","TimestampRange":{"StartTimestamp":1700000000,"EndTimestamp":1700000004}}}`)
	assertStatus(t, resp, http.StatusOK)
	var hls struct{ HLSStreamingSessionURL string }
	if err := json.Unmarshal(mustBody(t, resp), &hls); err != nil || !strings.HasPrefix(hls.HLSStreamingSessionURL, ts.URL+kvsHLSMasterPath) {
		t.Fatalf("HLS URL: %v %q", err, hls.HLSStreamingSessionURL)
	}
	status, ctype, master := kvsGetURL(t, hls.HLSStreamingSessionURL)
	if status != http.StatusOK || ctype != "application/x-mpegURL" || !strings.Contains(master, "getHLSMediaPlaylist.m3u8?SessionToken=") {
		t.Fatalf("master playlist: %d %s %q", status, ctype, master)
	}
	base, _ := url.Parse(hls.HLSStreamingSessionURL)
	mediaURL, _ := base.Parse(strings.TrimSpace(strings.Split(master, "\n")[3]))
	_, _, media := kvsGetURL(t, mediaURL.String())
	if strings.Count(media, "#EXTINF") != 3 || !strings.Contains(media, "#EXT-X-ENDLIST") {
		t.Fatalf("media playlist: %q", media)
	}
	var segment string
	for _, line := range strings.Split(media, "\n") {
		if strings.HasPrefix(line, "getKVSFragment.mkv") {
			segment = line
			break
		}
	}
	segURL, _ := base.Parse(segment)
	status, ctype, frag := kvsGetURL(t, segURL.String())
	if status != http.StatusOK || ctype != "video/webm" || !strings.Contains(frag, "AWS_KINESISVIDEO_FRAGMENT_NUMBER") {
		t.Fatalf("segment: %d %s", status, ctype)
	}

	resp = kvsMediaJSON(t, ts, "/getDASHStreamingSessionURL", `{"StreamName":"archive"}`)
	var dash struct{ DASHStreamingSessionURL string }
	if err := json.Unmarshal(mustBody(t, resp), &dash); err != nil || dash.DASHStreamingSessionURL == "" {
		t.Fatalf("DASH URL: %v", err)
	}
	status, ctype, mpd := kvsGetURL(t, dash.DASHStreamingSessionURL)
	if status != http.StatusOK || ctype != "application/dash+xml" || strings.Count(mpd, "<SegmentURL") != 4 {
		t.Fatalf("DASH manifest: %d %s %q", status, ctype, mpd)
	}
	// An HLS token can't open the DASH manifest.
	status, _, _ = kvsGetURL(t, ts.URL+kvsDASHPath+"?"+base.RawQuery)
	if status != http.StatusForbidden {
		t.Fatalf("cross-protocol token: %d", status)
	}

	for name, body := range map[string]string{
		"live with range":   `{"StreamName":"archive","HLSFragmentSelector":{"TimestampRange":{"StartTimestamp":1}}}`,
		"on demand no end":  `{"StreamName":"archive","PlaybackMode":"ON_DEMAND","HLSFragmentSelector":{"TimestampRange":{"StartTimestamp":1}}}`,
		"replay no range":   `{"StreamName":"archive","PlaybackMode":"LIVE_REPLAY"}`,
		"expires too short": `{"StreamName":"archive","Expires":10}`,
		"bad container":     `{"StreamName":"archive","ContainerFormat":"AVI"}`,
	} {
		resp = kvsMediaJSON(t, ts, "/getHLSStreamingSessionURL", body)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s: %d", name, resp.StatusCode)
		}
		resp.Body.Close()
	}
	// A window with no fragments has nothing to play.
	resp = kvsMediaJSON(t, ts, "/getHLSStreamingSessionURL", `{"StreamName":"archive","PlaybackMode":"ON_DEMAND","HLSFragmentSelector":{"FragmentSelectorType":"PRODUCER_TIMESTAMP","TimestampRange":{"StartTimestamp":1600000000,"EndTimestamp":1600000010}}}`)
	assertStatus(t, resp, http.StatusNotFound)
}
