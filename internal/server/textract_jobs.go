package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
	"github.com/stackyard/stackyard/internal/services/sns"
)

// Async jobs: Start* analyses the document at once and stores the result; the
// first Get* reports IN_PROGRESS and later reads report SUCCEEDED. Completion
// publishes to the NotificationChannel topic and writes OutputConfig, both
// after the store lock is released.
// ponytail: settle-on-read instead of background workers.

const (
	txMaxJobs     = 1000
	txJobPageSize = 1000
	// txMaxStoredBlocks bounds the analysis results held across all jobs
	// (roughly one block per word, line and page); the oldest jobs go first.
	txMaxStoredBlocks = 2_000_000
)

// txResult is a finished analysis. Its slices and maps are never mutated
// after the job is stored, so responses may read them after unlock.
type txResult struct {
	pages        int
	cost         int    // approximate stored blocks, for the store-wide budget
	itemKey      string // Blocks, ExpenseDocuments or Results
	items        []any
	summary      map[string]any // lending only
	modelKey     string
	modelVersion string
}

type txJob struct {
	id, api, tag    string
	created         time.Time
	reads           int
	result          *txResult
	topicARN        string
	outBucket, outP string
	hasOutput       bool
	location        map[string]any
	notified        bool
}

func txJobID() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// txParamHash fingerprints a request without its ClientRequestToken, for
// idempotency checks. json.Marshal sorts map keys, so it is canonical.
func txParamHash(in map[string]any) string {
	cp := make(map[string]any, len(in))
	for k, v := range in {
		if k != "ClientRequestToken" {
			cp[k] = v
		}
	}
	raw, _ := json.Marshal(cp)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// txStart validates the shared Start* inputs, reads the document and stores
// the job whose result compute builds.
func (s *Server) txStart(api string, in map[string]any, compute func(*txAnalysis) (*txResult, *txErr)) (map[string]any, *txErr) {
	loc := txMap(in, "DocumentLocation")
	obj := txMap(loc, "S3Object")
	if obj == nil {
		return nil, txInvalid("DocumentLocation needs an S3Object")
	}
	job := &txJob{api: api, tag: txStr(in, "JobTag"), location: obj}
	if nc := txMap(in, "NotificationChannel"); nc != nil {
		job.topicARN = txStr(nc, "SNSTopicArn")
		if _, err := s.sns.GetTopic(job.topicARN); err != nil {
			return nil, txInvalid("NotificationChannel topic %s does not exist", job.topicARN)
		}
	}
	if oc := txMap(in, "OutputConfig"); oc != nil {
		job.outBucket, job.outP, job.hasOutput = txStr(oc, "S3Bucket"), txStr(oc, "S3Prefix"), true
		if _, err := s.s3.GetBucket(job.outBucket); err != nil {
			return nil, txError("InvalidS3ObjectException", "OutputConfig bucket "+job.outBucket+" does not exist")
		}
	}
	token := txStr(in, "ClientRequestToken")
	hash := txParamHash(in)
	if id, err := s.textract.jobForToken(api, token, hash); err != nil || id != "" {
		if err != nil {
			return nil, err
		}
		return map[string]any{"JobId": id}, nil
	}
	raw, err := s.txS3Bytes(obj, txAsyncMaxBytes)
	if err != nil {
		return nil, err
	}
	doc, err := txParseDocument(raw)
	if err != nil {
		return nil, err
	}
	result, err := compute(txAnalyze(doc))
	if err != nil {
		return nil, err
	}
	result.pages = len(doc.pages)
	result.cost = doc.wordCount() + len(doc.pages)
	for _, lines := range doc.pages {
		result.cost += len(lines)
	}
	return map[string]any{"JobId": s.textract.addJob(job, result, token, hash)}, nil
}

func (s *Server) txStartDocumentTextDetection(in map[string]any) (map[string]any, *txErr) {
	return s.txStart("StartDocumentTextDetection", in, func(a *txAnalysis) (*txResult, *txErr) {
		return &txResult{itemKey: "Blocks", items: a.textBlocks(), modelKey: "DetectDocumentTextModelVersion", modelVersion: "1.0"}, nil
	})
}

func (s *Server) txStartDocumentAnalysis(in map[string]any) (map[string]any, *txErr) {
	features, err := txFeatures(in)
	if err != nil {
		return nil, err
	}
	if err := s.textract.checkAdapters(in); err != nil {
		return nil, err
	}
	return s.txStart("StartDocumentAnalysis", in, func(a *txAnalysis) (*txResult, *txErr) {
		blocks, err := a.analyzeBlocks(features, txQueries(in))
		if err != nil {
			return nil, err
		}
		return &txResult{itemKey: "Blocks", items: blocks, modelKey: "AnalyzeDocumentModelVersion", modelVersion: "1.0"}, nil
	})
}

func (s *Server) txStartExpenseAnalysis(in map[string]any) (map[string]any, *txErr) {
	return s.txStart("StartExpenseAnalysis", in, func(a *txAnalysis) (*txResult, *txErr) {
		var docs []any
		for _, p := range a.pages {
			docs = append(docs, a.expenseDocument(p.number, []*txPageInfo{p}))
		}
		return &txResult{itemKey: "ExpenseDocuments", items: docs, modelKey: "AnalyzeExpenseModelVersion", modelVersion: "1.0"}, nil
	})
}

func (s *Server) txStartLendingAnalysis(in map[string]any) (map[string]any, *txErr) {
	return s.txStart("StartLendingAnalysis", in, func(a *txAnalysis) (*txResult, *txErr) {
		return &txResult{itemKey: "Results", items: a.lendingResults(), summary: a.lendingSummary(), modelKey: "AnalyzeLendingModelVersion", modelVersion: "1.0"}, nil
	})
}

// txGetJob serves Get* for jobs started by api; summary serves
// GetLendingAnalysisSummary.
func txGetJob(api string, summary bool) txHandler {
	return func(s *Server, in map[string]any) (map[string]any, *txErr) {
		job, status, err := s.textract.readJob(api, txStr(in, "JobId"))
		if err != nil {
			return nil, err
		}
		out := map[string]any{"JobStatus": status}
		if status != "SUCCEEDED" {
			return out, nil
		}
		s.txComplete(job)
		r := job.result
		out["DocumentMetadata"] = map[string]any{"Pages": r.pages}
		out[r.modelKey] = r.modelVersion
		if summary {
			out["Summary"] = r.summary
			return out, nil
		}
		start, err := txPageToken(txStr(in, "NextToken"), len(r.items))
		if err != nil {
			return nil, err
		}
		size := txJobPageSize
		if n, ok := in["MaxResults"].(float64); ok && n >= 1 && n < float64(size) {
			size = int(n)
		}
		end := min(start+size, len(r.items))
		out[r.itemKey] = r.items[start:end]
		if end < len(r.items) {
			out["NextToken"] = "tx-" + strconv.Itoa(end)
		}
		return out, nil
	}
}

func txPageToken(token string, n int) (int, *txErr) {
	if token == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(strings.TrimPrefix(token, "tx-"), 10, 32)
	if err != nil || !strings.HasPrefix(token, "tx-") || v < 0 || int(v) > n {
		return 0, txInvalid("NextToken is invalid")
	}
	return int(v), nil
}

// txComplete publishes the completion notification and writes OutputConfig
// once per job. It runs without the store lock held.
func (s *Server) txComplete(job *txJob) {
	if !s.textract.markNotified(job) {
		return
	}
	if job.topicARN != "" {
		msg, _ := json.Marshal(map[string]any{
			"JobId": job.id, "Status": "SUCCEEDED", "API": job.api, "JobTag": job.tag,
			"Timestamp":        time.Now().UnixMilli(),
			"DocumentLocation": map[string]any{"S3ObjectName": job.location["Name"], "S3Bucket": job.location["Bucket"]},
		})
		_, _ = s.sns.Publish(sns.PublishInput{TopicARN: job.topicARN, Message: string(msg)})
	}
	if job.hasOutput {
		r := job.result
		full := map[string]any{"JobStatus": "SUCCEEDED", "DocumentMetadata": map[string]any{"Pages": r.pages}, r.itemKey: r.items, r.modelKey: r.modelVersion}
		getOp := map[string]string{
			"StartDocumentTextDetection": "GetDocumentTextDetection", "StartDocumentAnalysis": "GetDocumentAnalysis",
			"StartExpenseAnalysis": "GetExpenseAnalysis", "StartLendingAnalysis": "GetLendingAnalysis",
		}[job.api]
		model := awsmodels.TextractModel()
		body, _ := json.Marshal(model.Conform(model.Operations[getOp].Output, full))
		prefix := strings.Trim(job.outP, "/")
		if prefix == "" {
			prefix = "textract_output"
		}
		_, _ = s.s3.PutObject(job.outBucket, prefix+"/"+job.id+"/1", "application/json", body, nil, "", "")
	}
}
