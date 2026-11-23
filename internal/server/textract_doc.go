package server

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Textract has no OCR engine here (and may not add a dependency for one), so
// analysis is deterministic over text it can read without OCR:
//   - plain UTF-8 text documents (pages split on form feed),
//   - text-show operators in PDF content streams (uncompressed or FlateDecode),
//   - PNG / JPEG / TIFF images, which yield a PAGE block and no text.
// ponytail: no OCR; image text needs an OCR engine, which would be a new dependency.

const (
	txMaxPages        = 3000     // AWS's async page limit
	txMaxLinesPerPage = 2000     // bounds blocks per page
	txMaxWords        = 200000   // bounds blocks per document
	txMaxPDFInflate   = 32 << 20 // decompressed bytes across all PDF streams (bomb guard)
	txMaxPDFStreams   = 10000    // content streams scanned per PDF
	txSyncMaxBytes    = 10 << 20 // synchronous document limit (model ImageBlob max, AWS docs)
	txAsyncMaxBytes   = 32 << 20 // ponytail: AWS allows 500 MB async; 32 MiB bounds memory here
)

// txDocument is a parsed document: one entry per page, each a list of text lines.
type txDocument struct {
	format string // TEXT, PDF, PNG, JPEG, TIFF
	pages  [][]string
	seed   string // content hash; makes block ids stable per document
}

func (d *txDocument) wordCount() int {
	n := 0
	for _, lines := range d.pages {
		for _, l := range lines {
			n += len(txSplitWords(l))
		}
	}
	return n
}

// txParseDocument detects the format by magic bytes and extracts text.
func txParseDocument(b []byte) (*txDocument, *txErr) {
	if len(b) == 0 {
		return nil, txError("BadDocumentException", "Amazon Textract can't process the document because it is empty.")
	}
	sum := sha256.Sum256(b)
	doc := &txDocument{seed: hex.EncodeToString(sum[:8])}
	switch {
	case bytes.HasPrefix(b, []byte("%PDF-")):
		doc.format = "PDF"
		pages, err := txPDFPages(b)
		if err != nil {
			return nil, err
		}
		doc.pages = pages
	case bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")):
		doc.format, doc.pages = "PNG", [][]string{nil}
	case bytes.HasPrefix(b, []byte{0xFF, 0xD8, 0xFF}):
		doc.format, doc.pages = "JPEG", [][]string{nil}
	case bytes.HasPrefix(b, []byte("II*\x00")) || bytes.HasPrefix(b, []byte("MM\x00*")):
		doc.format, doc.pages = "TIFF", [][]string{nil}
	default:
		if !utf8.Valid(b) || bytes.IndexByte(b, 0) >= 0 {
			return nil, txError("UnsupportedDocumentException", "Request has unsupported document format")
		}
		doc.format = "TEXT"
		for _, page := range strings.Split(string(b), "\f") {
			doc.pages = append(doc.pages, txTextLines(page))
		}
	}
	if len(doc.pages) > txMaxPages {
		return nil, txError("DocumentTooLargeException", "Document exceeds the page limit of "+strconv.Itoa(txMaxPages))
	}
	for _, lines := range doc.pages {
		if len(lines) > txMaxLinesPerPage {
			return nil, txError("DocumentTooLargeException", "A page exceeds the limit of "+strconv.Itoa(txMaxLinesPerPage)+" lines")
		}
	}
	if doc.wordCount() > txMaxWords {
		return nil, txError("DocumentTooLargeException", "Document exceeds the limit of "+strconv.Itoa(txMaxWords)+" words")
	}
	return doc, nil
}

// txTextLines splits text into non-empty lines with trailing space trimmed.
func txTextLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimRightFunc(l, unicode.IsSpace)
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

// txWord is a word with rune offsets into its line.
type txWord struct {
	text       string
	start, end int
}

// txSplitWords splits a line on whitespace and table delimiters ('|', tab).
func txSplitWords(line string) []txWord {
	var out []txWord
	runes := []rune(line)
	start := -1
	for i := 0; i <= len(runes); i++ {
		sep := i == len(runes) || unicode.IsSpace(runes[i]) || runes[i] == '|'
		if sep && start >= 0 {
			out = append(out, txWord{text: string(runes[start:i]), start: start, end: i})
			start = -1
		} else if !sep && start < 0 {
			start = i
		}
	}
	return out
}

var (
	txPDFPageRe   = regexp.MustCompile(`/Type\s*/Page\b`)
	txPDFStreamRe = regexp.MustCompile(`stream\r?\n`)
)

// txPDFPages pulls text out of a PDF's content streams. When the number of
// text-bearing streams equals the page count they map one to one; otherwise
// all text lands on page 1.
// ponytail: no object-graph walk; a real PDF parser would map streams to pages exactly.
func txPDFPages(b []byte) ([][]string, *txErr) {
	pageCount := len(txPDFPageRe.FindAllIndex(b, txMaxPages+1))
	if pageCount == 0 {
		pageCount = 1
	}
	if pageCount > txMaxPages {
		return nil, txError("DocumentTooLargeException", "Document exceeds the page limit of "+strconv.Itoa(txMaxPages))
	}
	var streams [][]string
	budget := txMaxPDFInflate
	locs := txPDFStreamRe.FindAllIndex(b, txMaxPDFStreams)
	for _, loc := range locs {
		end := bytes.Index(b[loc[1]:], []byte("endstream"))
		if end < 0 {
			continue
		}
		data := b[loc[1] : loc[1]+end]
		dictStart := bytes.LastIndex(b[:loc[0]], []byte("obj"))
		if dictStart < 0 {
			dictStart = 0
		}
		if bytes.Contains(b[dictStart:loc[0]], []byte("/FlateDecode")) {
			zr, err := zlib.NewReader(bytes.NewReader(data))
			if err != nil {
				continue
			}
			inflated, err := io.ReadAll(io.LimitReader(zr, int64(budget)+1))
			_ = zr.Close()
			if err != nil && len(inflated) == 0 {
				continue
			}
			if len(inflated) > budget {
				return nil, txError("DocumentTooLargeException", "PDF content streams are too large to process")
			}
			budget -= len(inflated)
			data = inflated
		}
		if lines := txPDFLines(data); len(lines) > 0 {
			streams = append(streams, lines)
		}
	}
	pages := make([][]string, pageCount)
	if len(streams) == pageCount {
		copy(pages, streams)
		return pages, nil
	}
	for _, s := range streams {
		pages[0] = append(pages[0], s...)
	}
	return pages, nil
}

// txPDFLines reads text-show operators (Tj, TJ, ', ") from a content stream,
// breaking lines on text positioning operators (Td, TD, Tm, T*, ET).
func txPDFLines(c []byte) []string {
	var lines, pend []string
	var cur strings.Builder
	flush := func() {
		if s := strings.TrimSpace(cur.String()); s != "" {
			lines = append(lines, s)
		}
		cur.Reset()
	}
	inArray := false
	for i := 0; i < len(c); {
		ch := c[i]
		switch {
		case ch == '(':
			s, n := txPDFLiteral(c[i:])
			pend = append(pend, s)
			i += n
		case ch == '<' && i+1 < len(c) && c[i+1] != '<':
			end := bytes.IndexByte(c[i:], '>')
			if end < 0 {
				return lines
			}
			if raw, err := hex.DecodeString(strings.Map(txHexOnly, string(c[i+1:i+end]))); err == nil {
				pend = append(pend, txLatin1(raw))
			}
			i += end + 1
		case ch == '[':
			inArray = true
			i++
		case ch == ']':
			inArray = false
			i++
		case ch == '%':
			for i < len(c) && c[i] != '\n' && c[i] != '\r' {
				i++
			}
		case inArray && (ch == '-' || ch == '.' || (ch >= '0' && ch <= '9')):
			j := i + 1
			for j < len(c) && (c[j] == '.' || (c[j] >= '0' && c[j] <= '9')) {
				j++
			}
			// A large negative TJ adjustment is a word gap.
			if v, err := strconv.ParseFloat(string(c[i:j]), 64); err == nil && v <= -200 {
				pend = append(pend, " ")
			}
			i = j
		case (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '\'' || ch == '"' || ch == '*':
			j := i + 1
			for j < len(c) && ((c[j] >= 'A' && c[j] <= 'Z') || (c[j] >= 'a' && c[j] <= 'z') || c[j] == '*') {
				j++
			}
			switch string(c[i:j]) {
			case "Tj", "TJ":
				cur.WriteString(strings.Join(pend, ""))
			case "'", "\"":
				flush()
				cur.WriteString(strings.Join(pend, ""))
			case "T*", "Td", "TD", "Tm", "ET":
				flush()
			}
			pend = nil
			i = j
		default:
			i++
		}
	}
	flush()
	return lines
}

// txPDFLiteral decodes a PDF literal string starting at c[0] == '(' and
// returns it with the number of bytes consumed.
func txPDFLiteral(c []byte) (string, int) {
	var out []byte
	depth := 0
	for i := 0; i < len(c); i++ {
		ch := c[i]
		switch {
		case ch == '\\' && i+1 < len(c):
			i++
			switch e := c[i]; e {
			case 'n':
				out = append(out, '\n')
			case 'r':
				out = append(out, '\r')
			case 't':
				out = append(out, '\t')
			case 'b', 'f':
			case '\r', '\n':
			default:
				if e >= '0' && e <= '7' {
					v, j := 0, i
					for ; j < len(c) && j < i+3 && c[j] >= '0' && c[j] <= '7'; j++ {
						v = v*8 + int(c[j]-'0')
					}
					out = append(out, byte(v))
					i = j - 1
				} else {
					out = append(out, e)
				}
			}
		case ch == '(':
			if depth > 0 {
				out = append(out, ch)
			}
			depth++
		case ch == ')':
			depth--
			if depth == 0 {
				return txLatin1(out), i + 1
			}
			out = append(out, ch)
		default:
			out = append(out, ch)
		}
	}
	return txLatin1(out), len(c)
}

func txHexOnly(r rune) rune {
	if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
		return r
	}
	return -1
}

// txLatin1 maps PDF string bytes to runes (PDFDocEncoding ≈ Latin-1 for text).
func txLatin1(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	r := make([]rune, len(b))
	for i, c := range b {
		r[i] = rune(c)
	}
	return string(r)
}
