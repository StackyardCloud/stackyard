package server

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"path"
	"sort"
	"strconv"
	"strings"
)

// Amazon ML data paths. S3 is read before the store lock is taken and
// written after it is released; the two stores' locks never nest.

const (
	amlMaxSchemaChars = 131071
	// amlMaxBatchRows caps the rows one batch prediction scores, so bounded
	// input can't turn into unbounded output.
	amlMaxBatchRows = 100000
)

// amlPre is what a create operation learned from S3 before locking.
type amlPre struct {
	inputErr *fcErr // the request itself is invalid
	failure  string // the resource settles FAILED with this message
	schema   string
	header   bool
	size     int64
	files    int
	records  int
	// batch predictions
	outKey, outBucket string
	outBody           []byte
}

type amlSchema struct {
	TargetAttributeName    string `json:"targetAttributeName"`
	DataFileContainsHeader bool   `json:"dataFileContainsHeader"`
}

func amlParseSchema(raw string) (amlSchema, error) {
	var sc amlSchema
	if err := json.Unmarshal([]byte(raw), &sc); err != nil {
		return sc, fmt.Errorf("the data schema is not a JSON object: %w", err)
	}
	return sc, nil
}

// amlCountRecords counts non-empty lines, less one header line per file.
func amlCountRecords(files []pszS3File, header bool) int {
	n := 0
	for _, f := range files {
		lines := bytes.Count(f.body, []byte("\n"))
		if len(f.body) > 0 && f.body[len(f.body)-1] != '\n' {
			lines++
		}
		if header && lines > 0 {
			lines--
		}
		n += lines
	}
	return n
}

// amlReadSource reads an S3 data source's schema and data.
func (s *Server) amlReadSource(spec map[string]any) amlPre {
	var pre amlPre
	pre.schema = fcStr(spec, "DataSchema")
	if pre.schema == "" {
		loc := fcStr(spec, "DataSchemaLocationS3")
		if loc == "" {
			pre.inputErr = fcInvalid("DataSpec needs DataSchema or DataSchemaLocationS3")
			return pre
		}
		files, err := s.pszReadS3(loc)
		if err != nil || len(files) == 0 {
			pre.failure = fmt.Sprintf("The schema at %s can't be read.", loc)
			return pre
		}
		pre.schema = string(files[0].body)
		if len(pre.schema) > amlMaxSchemaChars {
			pre.failure = fmt.Sprintf("The schema at %s is larger than %d characters.", loc, amlMaxSchemaChars)
			pre.schema = ""
			return pre
		}
	}
	sc, err := amlParseSchema(pre.schema)
	if err != nil {
		if fcStr(spec, "DataSchema") != "" {
			pre.inputErr = fcInvalid("%s", err.Error())
			return pre
		}
		pre.failure, pre.schema = err.Error(), ""
		return pre
	}
	pre.header = sc.DataFileContainsHeader
	files, err := s.pszReadS3(fcStr(spec, "DataLocationS3"))
	if err != nil {
		pre.failure = err.Error()
		return pre
	}
	for _, f := range files {
		pre.size += int64(len(f.body))
	}
	pre.files, pre.records = len(files), amlCountRecords(files, pre.header)
	return pre
}

// amlBatchSnapshot is what a batch prediction needs from the store before
// it reads S3: the data source's location and layout and the model's type.
type amlBatchSnapshot struct {
	location, modelType string
	header              bool
	threshold           float64
}

// amlPrepareBatch reads the batch's input and scores it, outside the lock.
func (s *Server) amlPrepareBatch(in map[string]any) amlPre {
	var pre amlPre
	snap, ok := s.machineLearning.batchSnapshot(fcStr(in, "MLModelId"), fcStr(in, "BatchPredictionDataSourceId"))
	if !ok {
		return pre // the create reports the missing resource
	}
	outURI := fcStr(in, "OutputUri")
	if problem := s.pszOutputProblem(outURI); problem != "" {
		pre.failure = problem
		return pre
	}
	files, err := s.pszReadS3(snap.location)
	if err != nil {
		pre.failure = err.Error()
		return pre
	}
	var out bytes.Buffer
	rows := 0
	for _, f := range files {
		lines := strings.Split(strings.TrimRight(string(f.body), "\n"), "\n")
		if snap.header && len(lines) > 0 {
			lines = lines[1:]
		}
		for _, line := range lines {
			if rows >= amlMaxBatchRows {
				break
			}
			if line == "" {
				continue
			}
			out.WriteString(amlBatchLine(snap, line))
			rows++
		}
	}
	bucket, prefix, _ := pszS3Path(outURI)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	name := "data"
	if len(files) > 0 {
		name = path.Base(files[0].key)
	}
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	_, _ = zw.Write([]byte(amlBatchHeader(snap.modelType)))
	_, _ = zw.Write(out.Bytes())
	_ = zw.Close()
	pre.records, pre.outBucket, pre.outBody = rows, bucket, gz.Bytes()
	pre.outKey = prefix + "batch-prediction/result/" + fcStr(in, "BatchPredictionId") + "-" + name + ".gz"
	return pre
}

func amlBatchHeader(modelType string) string {
	switch modelType {
	case "BINARY":
		return "bestAnswer,score\n"
	case "MULTICLASS":
		return strings.Join(amlClasses, ",") + "\n"
	}
	return "score\n"
}

func amlBatchLine(snap amlBatchSnapshot, line string) string {
	score := amlScore(line)
	switch snap.modelType {
	case "BINARY":
		best := 0
		if score >= snap.threshold {
			best = 1
		}
		return fmt.Sprintf("%d,%s\n", best, amlFmt(score))
	case "MULTICLASS":
		scores := amlClassScores(line)
		parts := make([]string, len(amlClasses))
		for i, c := range amlClasses {
			parts[i] = amlFmt(scores[c])
		}
		return strings.Join(parts, ",") + "\n"
	}
	return amlFmt(score*100) + "\n"
}

// ponytail: placeholder model — scores are a stable hash of the input, the
// shape (labels, ranges, thresholds) is what Amazon ML returns.
var amlClasses = []string{"0", "1", "2"}

func amlScore(key string) float64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	return float64(h.Sum64()%1_000_000) / 1_000_000
}

func amlClassScores(key string) map[string]float64 {
	out := map[string]float64{}
	total := 0.0
	for _, c := range amlClasses {
		out[c] = amlScore(c+"|"+key) + 0.01
		total += out[c]
	}
	for c := range out {
		out[c] = math.Round(out[c]/total*1e6) / 1e6
	}
	return out
}

func amlFmt(f float64) string { return strconv.FormatFloat(f, 'f', 6, 64) }

// amlRecordKey is a stable rendering of a Predict record.
func amlRecordKey(rec map[string]any) string {
	keys := make([]string, 0, len(rec))
	for k := range rec {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		v, _ := rec[k].(string)
		b.WriteString(k + "=" + v + "\x00")
	}
	return b.String()
}

// amlPredict scores one record for a model of modelType.
func amlPredict(modelType string, threshold float64, rec map[string]any) map[string]any {
	key := amlRecordKey(rec)
	details := map[string]any{"PredictiveModelType": modelType, "Algorithm": "SGD"}
	score := amlScore(key)
	switch modelType {
	case "BINARY":
		label := "0"
		if score >= threshold {
			label = "1"
		}
		return map[string]any{"predictedLabel": label, "predictedScores": map[string]any{label: score}, "details": details}
	case "MULTICLASS":
		scores := amlClassScores(key)
		best, out := amlClasses[0], map[string]any{}
		for _, c := range amlClasses {
			out[c] = scores[c]
			if scores[c] > scores[best] {
				best = c
			}
		}
		return map[string]any{"predictedLabel": best, "predictedScores": out, "details": details}
	}
	return map[string]any{"predictedValue": math.Round(score*100*1e4) / 1e4, "details": details}
}

func (s *Server) amlRun(op string, in map[string]any, base string) (any, *fcErr) {
	var pre amlPre
	switch op {
	case "CreateDataSourceFromS3":
		pre = s.amlReadSource(fcObj(in, "DataSpec"))
		if pre.inputErr != nil {
			return nil, pre.inputErr
		}
	case "CreateBatchPrediction":
		pre = s.amlPrepareBatch(in)
	}
	out, write, err := s.machineLearning.run(op, in, base, &pre)
	if err != nil {
		return nil, err
	}
	if write && pre.outBody != nil {
		_, _ = s.s3.PutObject(pre.outBucket, pre.outKey, "application/gzip", pre.outBody, nil, "", "")
	}
	return out, nil
}
