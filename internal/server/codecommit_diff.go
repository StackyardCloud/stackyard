package server

import (
	"strings"
	"unicode/utf8"
)

// ccMaxDiffLines bounds the LCS table (lines × lines). Larger files diff as
// a whole-file replacement.
// ponytail: O(n·m) LCS; switch to Myers if big text diffs matter.
const ccMaxDiffLines = 2000

type ccDiffLine struct {
	kind          string // CONTEXT | ADD | DELETE
	before, after int    // 1-based line numbers (0 when absent)
	content       string
}

func ccLineDiff(a, b []string) []ccDiffLine {
	var out []ccDiffLine
	if len(a) > ccMaxDiffLines || len(b) > ccMaxDiffLines {
		for i, l := range a {
			out = append(out, ccDiffLine{kind: "DELETE", before: i + 1, content: l})
		}
		for j, l := range b {
			out = append(out, ccDiffLine{kind: "ADD", after: j + 1, content: l})
		}
		return out
	}
	// lcs[i][j] = LCS length of a[i:] and b[j:]
	lcs := make([][]int32, len(a)+1)
	for i := range lcs {
		lcs[i] = make([]int32, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		switch {
		case i < len(a) && j < len(b) && a[i] == b[j]:
			out = append(out, ccDiffLine{kind: "CONTEXT", before: i + 1, after: j + 1, content: a[i]})
			i, j = i+1, j+1
		case j < len(b) && (i == len(a) || lcs[i][j+1] >= lcs[i+1][j]):
			out = append(out, ccDiffLine{kind: "ADD", after: j + 1, content: b[j]})
			j++
		default:
			out = append(out, ccDiffLine{kind: "DELETE", before: i + 1, content: a[i]})
			i++
		}
	}
	return out
}

// ccDiffHunks groups changed lines with up to context lines around them.
func ccDiffHunks(lines []ccDiffLine, context int) []map[string]any {
	var hunks []map[string]any
	for i := 0; i < len(lines); {
		if lines[i].kind == "CONTEXT" {
			i++
			continue
		}
		start := max(0, i-context)
		end := i
		for end < len(lines) {
			if lines[end].kind != "CONTEXT" {
				end++
				continue
			}
			// a run of context: stop if it is longer than 2*context
			run := end
			for run < len(lines) && lines[run].kind == "CONTEXT" {
				run++
			}
			if run == len(lines) || run-end > 2*context {
				end = min(end+context, len(lines))
				break
			}
			end = run
		}
		changes := []any{}
		beforeStart, afterStart, beforeCount, afterCount := 0, 0, 0, 0
		for _, l := range lines[start:end] {
			c := map[string]any{"type": l.kind, "content": strings.TrimSuffix(l.content, "\n")}
			if l.before > 0 {
				c["beforeLineNumber"] = l.before
				beforeCount++
				if beforeStart == 0 {
					beforeStart = l.before
				}
			}
			if l.after > 0 {
				c["afterLineNumber"] = l.after
				afterCount++
				if afterStart == 0 {
					afterStart = l.after
				}
			}
			changes = append(changes, c)
		}
		hunks = append(hunks, map[string]any{"beforeStartLine": beforeStart, "beforeLineCount": beforeCount, "afterStartLine": afterStart, "afterLineCount": afterCount, "changes": changes})
		i = end
	}
	return hunks
}

func (st *ccStore) getBlobDifferences(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	blob := func(key string) ([]byte, *ccErr) {
		id := ccStr(in, key)
		if id == "" {
			return nil, nil
		}
		if len(id) != 40 {
			return nil, ccError("InvalidBlobIdException", "blob id "+id+" is not valid")
		}
		b, ok := r.blobs[id]
		if !ok {
			return nil, ccError("BlobIdDoesNotExistException", "blob "+id+" does not exist")
		}
		return b, nil
	}
	before, err := blob("beforeBlobId")
	if err != nil {
		return nil, err
	}
	after, err := blob("afterBlobId")
	if err != nil {
		return nil, err
	}
	out := map[string]any{"afterBlobSize": len(after), "beforeBlobSize": len(before)}
	if !utf8.Valid(before) || !utf8.Valid(after) {
		out["isBinary"], out["hunks"] = true, []any{}
		return out, nil
	}
	context := 3
	if v, ok := in["contextLines"].(float64); ok && v >= 0 && v <= 1000 {
		context = int(v)
	}
	a, b := ccSplitLines(before), ccSplitLines(after)
	if ccBool(in, "ignoreWhitespace") {
		a, b = ccTrimSpace(a), ccTrimSpace(b)
	}
	hunks := ccDiffHunks(ccLineDiff(a, b), context)
	page, next, perr := ccPage(hunks, in, "NextToken", "MaxResults")
	if perr != nil {
		return nil, perr
	}
	out["isBinary"], out["hunks"], out["NextToken"] = false, page, next
	return out, nil
}

func ccTrimSpace(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = strings.Join(strings.Fields(l), " ")
	}
	return out
}
