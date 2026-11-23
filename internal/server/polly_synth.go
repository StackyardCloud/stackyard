package server

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// pollyLimits are the text limits for one synthesis request.
type pollyLimits struct{ billed, total int }

var (
	pollySpeechLimits = pollyLimits{billed: 3000, total: 6000}
	pollyTaskLimits   = pollyLimits{billed: 100000, total: 200000}
)

const (
	// pollyMsPerChar paces the generated audio and speech-mark times.
	pollyMsPerChar = 60
	// pollyMaxAudioMs caps generated audio so output size stays bounded.
	// ponytail: the audio is silence sized to the text, not real speech; a
	// 60 s ceiling keeps the largest PCM response around 2 MB.
	pollyMaxAudioMs            = 60_000
	pollyMaxLexiconsPerRequest = 5
)

// pollySynthesis is a validated, rendered synthesis result.
type pollySynthesis struct {
	format string
	chars  int
	audio  []byte
}

// pollySynthesize validates a SynthesizeSpeech-shaped request and renders it.
func (s *Server) pollySynthesize(in map[string]any, limits pollyLimits) (*pollySynthesis, *pollyError) {
	format := pollyStr(in, "OutputFormat")
	voice, ok := pollyVoiceByID[pollyStr(in, "VoiceId")]
	if !ok {
		return nil, pollyValidation("VoiceId %s is not a valid voice.", pollyStr(in, "VoiceId"))
	}
	engine := pollyStr(in, "Engine")
	if engine == "" {
		engine = "standard"
	}
	if !voice.supports(engine) {
		return nil, pollyBad("EngineNotSupportedException", "This voice does not support the selected engine: %s", engine)
	}
	if lang := pollyStr(in, "LanguageCode"); lang != "" && !voice.speaks(lang) {
		return nil, pollyBad("LanguageNotSupportedException", "Voice %s does not support the language %s.", voice.id, lang)
	}
	lexicons := pollyStrings(in, "LexiconNames")
	if len(lexicons) > pollyMaxLexiconsPerRequest {
		return nil, pollyBad("MaxLexiconsNumberExceededException", "At most %d lexicons can be applied to one request.", pollyMaxLexiconsPerRequest)
	}
	if missing := s.polly.missingLexicon(lexicons); missing != "" {
		return nil, pollyErr(http.StatusNotFound, "LexiconNotFoundException", "Lexicon not found: %s", missing)
	}
	rate, err := pollySampleRate(format, engine, pollyStr(in, "SampleRate"))
	if err != nil {
		return nil, err
	}
	text, textType := pollyStr(in, "Text"), pollyStr(in, "TextType")
	doc, err := pollyParseText(text, textType)
	if err != nil {
		return nil, err
	}
	if utf8.RuneCountInString(text) > limits.total || doc.billed > limits.billed {
		return nil, pollyBad("TextLengthExceededException", "Maximum text length has been exceeded: %d billed and %d total characters are allowed.", limits.billed, limits.total)
	}
	marks := pollyStrings(in, "SpeechMarkTypes")
	switch {
	case format == "json" && len(marks) == 0:
		return nil, pollyValidation("SpeechMarkTypes is required when OutputFormat is json.")
	case format != "json" && len(marks) > 0:
		return nil, pollyBad("MarksNotSupportedForFormatException", "Speech marks are only supported with the json output format.")
	}
	for _, m := range marks {
		if m == "ssml" && textType != "ssml" {
			return nil, pollyBad("SsmlMarksNotSupportedForTextTypeException", "SSML speech marks are only supported for SSML input.")
		}
	}
	res := &pollySynthesis{format: format, chars: doc.billed}
	ms := min(max(doc.billed*pollyMsPerChar, 100), pollyMaxAudioMs)
	switch format {
	case "json":
		res.audio = pollySpeechMarks(doc, marks)
	case "pcm":
		res.audio = make([]byte, rate*2*ms/1000) // 16-bit mono silence
	case "mulaw":
		res.audio = bytes.Repeat([]byte{0xFF}, 8*ms) // mu-law silence
	case "alaw":
		res.audio = bytes.Repeat([]byte{0xD5}, 8*ms) // A-law silence
	case "mp3":
		res.audio = pollyMP3(rate, ms)
	default: // ogg_vorbis, ogg_opus
		res.audio = pollyOggOpus(rate, ms)
	}
	return res, nil
}

// pollySampleRate checks SampleRate for the format and returns the rate used.
func pollySampleRate(format, engine, raw string) (int, *pollyError) {
	allowed := map[string][]int{
		"pcm": {8000, 16000}, "mulaw": {8000}, "alaw": {8000},
		"mp3": {8000, 16000, 22050, 24000}, "ogg_vorbis": {8000, 16000, 22050, 24000},
		"ogg_opus": {8000, 16000, 24000, 48000}, "json": nil,
	}[format]
	def := map[string]int{"pcm": 16000, "mulaw": 8000, "alaw": 8000, "ogg_opus": 48000}[format]
	if def == 0 {
		def = 22050
		if engine != "standard" {
			def = 24000
		}
	}
	if raw == "" || format == "json" {
		return def, nil
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err == nil {
		for _, a := range allowed {
			if int(n) == a {
				return a, nil
			}
		}
	}
	return 0, pollyBad("InvalidSampleRateException", "The specified sample rate %s is not valid for the %s output format.", raw, format)
}

// pollyDoc is input text split into the spans speech marks are built from.
type pollyDoc struct {
	input  string
	ssml   bool
	billed int
	spans  []pollySpan // runs of spoken text with their byte offset in input
	marks  []pollyMark // <mark/> tags
}

type pollySpan struct {
	text   string
	offset int
	before int // billed characters before this span
}

type pollyMark struct {
	name   string
	offset int
	before int
}

// pollyParseText reads plain text or SSML. SSML must be well-formed XML with
// a <speak> root; only its character data is spoken and billed.
func pollyParseText(text, textType string) (*pollyDoc, *pollyError) {
	doc := &pollyDoc{input: text, ssml: textType == "ssml"}
	if !doc.ssml {
		doc.billed = utf8.RuneCountInString(text)
		doc.spans = []pollySpan{{text: text}}
		return doc, nil
	}
	dec := xml.NewDecoder(strings.NewReader(text))
	depth, sawRoot := 0, false
	for {
		start := int(dec.InputOffset())
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, pollyBad("InvalidSsmlException", "Unsupported SSML: %v", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if depth == 0 {
				if sawRoot || t.Name.Local != "speak" {
					return nil, pollyBad("InvalidSsmlException", "SSML must have a single <speak> root element.")
				}
				sawRoot = true
			}
			if t.Name.Local == "mark" {
				for _, a := range t.Attr {
					if a.Name.Local == "name" {
						doc.marks = append(doc.marks, pollyMark{name: a.Value, offset: start, before: doc.billed})
					}
				}
			}
			depth++
		case xml.EndElement:
			depth--
		case xml.CharData:
			if depth == 0 {
				if strings.TrimSpace(string(t)) != "" {
					return nil, pollyBad("InvalidSsmlException", "Text outside the <speak> element is not allowed.")
				}
				continue
			}
			doc.spans = append(doc.spans, pollySpan{text: string(t), offset: start, before: doc.billed})
			doc.billed += utf8.RuneCount(t)
		}
	}
	if !sawRoot {
		return nil, pollyBad("InvalidSsmlException", "SSML must have a <speak> root element.")
	}
	return doc, nil
}

var pollyWordRE = regexp.MustCompile(`\S+`)

// pollySpeechMarks renders newline-delimited JSON speech marks. Times follow
// the same per-character pacing as the generated audio.
func pollySpeechMarks(doc *pollyDoc, types []string) []byte {
	want := map[string]bool{}
	for _, t := range types {
		want[t] = true
	}
	type word struct {
		value      string
		start, end int
		time       int
	}
	var words []word
	for _, sp := range doc.spans {
		for _, loc := range pollyWordRE.FindAllStringIndex(sp.text, -1) {
			raw := sp.text[loc[0]:loc[1]]
			trimmed := strings.TrimFunc(raw, unicode.IsPunct)
			if trimmed == "" {
				continue
			}
			lead := strings.Index(raw, trimmed)
			start := sp.offset + loc[0] + lead
			words = append(words, word{
				value: trimmed, start: start, end: start + len(trimmed),
				time: (sp.before + utf8.RuneCountInString(sp.text[:loc[0]])) * pollyMsPerChar,
			})
		}
	}
	var out bytes.Buffer
	emit := func(v map[string]any) {
		raw, _ := json.Marshal(v)
		out.Write(raw)
		out.WriteByte('\n')
	}
	marks := doc.marks
	sentenceStart := 0
	for i, w := range words {
		for len(marks) > 0 && marks[0].before*pollyMsPerChar <= w.time {
			if want["ssml"] {
				m := marks[0]
				emit(map[string]any{"time": m.before * pollyMsPerChar, "type": "ssml", "start": m.offset, "end": m.offset, "value": m.name})
			}
			marks = marks[1:]
		}
		if i == sentenceStart && want["sentence"] {
			end := i
			for end < len(words)-1 && !pollySentenceEnd(doc.input, words[end].end) {
				end++
			}
			parts := make([]string, 0, end-i+1)
			for _, sw := range words[i : end+1] {
				parts = append(parts, sw.value)
			}
			value := strings.Join(parts, " ")
			if !doc.ssml {
				value = doc.input[w.start:min(words[end].end+1, len(doc.input))]
				value = strings.TrimSpace(value)
			}
			emit(map[string]any{"time": w.time, "type": "sentence", "start": w.start, "end": words[end].end, "value": value})
		}
		if pollySentenceEnd(doc.input, w.end) {
			sentenceStart = i + 1
		}
		if want["word"] {
			emit(map[string]any{"time": w.time, "type": "word", "start": w.start, "end": w.end, "value": w.value})
		}
		if want["viseme"] {
			emit(map[string]any{"time": w.time, "type": "viseme", "value": pollyViseme(w.value)})
		}
	}
	for _, m := range marks {
		if want["ssml"] {
			emit(map[string]any{"time": m.before * pollyMsPerChar, "type": "ssml", "start": m.offset, "end": m.offset, "value": m.name})
		}
	}
	return out.Bytes()
}

// pollySentenceEnd reports whether the word ending at byte end closes a
// sentence (it's followed by . ! or ?).
func pollySentenceEnd(input string, end int) bool {
	return end < len(input) && strings.ContainsRune(".!?", rune(input[end]))
}

// pollyViseme maps a word's first letter to a Polly viseme.
// ponytail: one viseme per word from its first letter; real visemes follow phonemes.
func pollyViseme(word string) string {
	r, _ := utf8.DecodeRuneInString(strings.ToLower(word))
	switch {
	case strings.ContainsRune("pbm", r):
		return "p"
	case strings.ContainsRune("fv", r):
		return "f"
	case strings.ContainsRune("tdnl", r):
		return "t"
	case strings.ContainsRune("sz", r):
		return "s"
	case strings.ContainsRune("kgqc", r):
		return "k"
	case r == 'r':
		return "r"
	case strings.ContainsRune("aeiou", r):
		return string(r)
	}
	return "sil"
}

// pollyMP3 builds silent MPEG-2 (or MPEG-2.5 at 8 kHz) Layer III frames at
// 32 kbps mono: a valid, decodable stream of the requested length.
func pollyMP3(rate, ms int) []byte {
	version, srIndex := byte(0xF3), byte(0) // MPEG-2, Layer III, no CRC
	switch rate {
	case 24000:
		srIndex = 1
	case 16000:
		srIndex = 2
	case 8000:
		version, srIndex = 0xE3, 2 // MPEG-2.5
	}
	frameLen := 72 * 32000 / rate
	frames := max(1, (rate*ms/1000+575)/576)
	out := make([]byte, 0, frames*frameLen)
	frame := make([]byte, frameLen)
	frame[0], frame[1], frame[2], frame[3] = 0xFF, version, 0x40|srIndex<<2, 0xC0 // 32 kbps, mono
	for i := 0; i < frames; i++ {
		out = append(out, frame...)
	}
	return out
}

// pollyOggOpus builds an Ogg Opus stream of 20 ms silence packets.
// ponytail: ogg_vorbis also gets Opus-coded silence in an Ogg container (still
// a valid audio/ogg file); generating a Vorbis setup header isn't worth it here.
func pollyOggOpus(rate, ms int) []byte {
	head := []byte("OpusHead")
	head = append(head, 1, 1)                          // version, mono
	head = binary.LittleEndian.AppendUint16(head, 312) // pre-skip
	head = binary.LittleEndian.AppendUint32(head, uint32(rate))
	head = append(head, 0, 0, 0) // gain, mapping family
	tags := append([]byte("OpusTags"), 9, 0, 0, 0)
	tags = append(tags, "stackyard"...)
	tags = append(tags, 0, 0, 0, 0)

	var out bytes.Buffer
	seq := uint32(0)
	page := func(flags byte, granule uint64, packets [][]byte) {
		hdr := []byte("OggS")
		hdr = append(hdr, 0, flags)
		hdr = binary.LittleEndian.AppendUint64(hdr, granule)
		hdr = binary.LittleEndian.AppendUint32(hdr, 0x5354594B) // stream serial
		hdr = binary.LittleEndian.AppendUint32(hdr, seq)
		hdr = append(hdr, 0, 0, 0, 0, byte(len(packets)))
		var body []byte
		for _, p := range packets {
			hdr = append(hdr, byte(len(p))) // every packet here is < 255 bytes
			body = append(body, p...)
		}
		pg := append(hdr, body...)
		binary.LittleEndian.PutUint32(pg[22:], pollyOggCRC(pg))
		out.Write(pg)
		seq++
	}
	page(0x02, 0, [][]byte{head})
	page(0, 0, [][]byte{tags})
	silence := []byte{0xF8, 0xFF, 0xFE} // CELT fullband 20 ms silence frame
	total := max(1, ms/20)
	for done := 0; done < total; {
		n := min(50, total-done)
		packets := make([][]byte, n)
		for i := range packets {
			packets[i] = silence
		}
		done += n
		flags := byte(0)
		if done == total {
			flags = 0x04
		}
		page(flags, uint64(312+done*960), packets)
	}
	return out.Bytes()
}

var pollyOggTable = func() (t [256]uint32) {
	for i := range t {
		r := uint32(i) << 24
		for j := 0; j < 8; j++ {
			if r&0x80000000 != 0 {
				r = r<<1 ^ 0x04C11DB7
			} else {
				r <<= 1
			}
		}
		t[i] = r
	}
	return
}()

// pollyOggCRC is the Ogg page checksum (CRC-32, poly 0x04C11DB7, unreflected).
func pollyOggCRC(page []byte) uint32 {
	var crc uint32
	for _, b := range page {
		crc = crc<<8 ^ pollyOggTable[byte(crc>>24)^b]
	}
	return crc
}

// writePollyAudio writes a synthesis result. The content type is chosen from
// a fixed set by output format, in the same function that writes the body.
func writePollyAudio(w http.ResponseWriter, format string, chars int, audio []byte) {
	h := w.Header()
	switch format {
	case "mp3":
		h.Set("Content-Type", "audio/mpeg")
	case "ogg_vorbis", "ogg_opus":
		h.Set("Content-Type", "audio/ogg")
	case "pcm":
		h.Set("Content-Type", "audio/pcm")
	case "mulaw":
		h.Set("Content-Type", "audio/basic")
	case "alaw":
		h.Set("Content-Type", "audio/x-alaw-basic")
	default:
		h.Set("Content-Type", "application/x-json-stream")
	}
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("x-amzn-RequestCharacters", strconv.Itoa(chars))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(audio)
}

// pollyAudioExt is the S3 object suffix synthesis tasks use per format.
func pollyAudioExt(format string) string {
	switch format {
	case "mp3":
		return "mp3"
	case "ogg_vorbis", "ogg_opus":
		return "ogg"
	case "json":
		return "marks"
	}
	return format
}
