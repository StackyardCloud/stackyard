package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"
)

const pszServicePrincipal = "personalize.amazonaws.com"

// Import, export, data deletion and batch jobs. S3 is read before the
// Personalize lock is taken and written after it is released; the two
// stores' locks never nest. Job effects apply at creation; the job still
// reports CREATE IN_PROGRESS (IN_PROGRESS for deletions) on its first read.

const (
	pszMaxInputBytes = 8 << 20
	pszMaxBatchLines = 10000
	// pszMaxOutputBytes caps the total size of a batch job's output, so a
	// large numResults can't amplify the bounded input into unbounded work.
	pszMaxOutputBytes = 8 << 20
)

type pszS3File struct {
	key  string
	body []byte
}

// pszS3Path splits s3://bucket/key.
func pszS3Path(p string) (string, string, error) {
	rest, ok := strings.CutPrefix(p, "s3://")
	bucket, key, _ := strings.Cut(rest, "/")
	if !ok || bucket == "" {
		return "", "", fmt.Errorf("%s isn't an s3:// path", p)
	}
	return bucket, key, nil
}

// pszReadS3 reads the object at p, or every object under the prefix p.
func (s *Server) pszReadS3(c s3Caller, p string) ([]pszS3File, error) {
	return s.pszReadS3As(c, pszServicePrincipal, p)
}

// pszReadS3As is pszReadS3 for another service reading on the caller's behalf.
func (s *Server) pszReadS3As(c s3Caller, principal, p string) ([]pszS3File, error) {
	bucket, key, err := pszS3Path(p)
	if err != nil {
		return nil, err
	}
	if obj, err := s.s3.GetObject(bucket, key); err == nil && key != "" && !strings.HasSuffix(key, "/") &&
		!s.authorizeServiceS3As(c, principal, bucket, key, "s3:GetObject") {
		return nil, fmt.Errorf("access denied reading s3://%s/%s", obj.Bucket, obj.Key)
	}
	if !s.authorizeServiceS3As(c, principal, bucket, "", "s3:ListBucket") {
		return nil, fmt.Errorf("access denied listing %s", p)
	}
	if obj, err := s.s3.GetObject(bucket, key); err == nil && key != "" && !strings.HasSuffix(key, "/") {
		if len(obj.Body) > pszMaxInputBytes {
			return nil, fmt.Errorf("%s is larger than the %d MiB Stackyard reads", p, pszMaxInputBytes>>20)
		}
		return []pszS3File{{obj.Key, obj.Body}}, nil
	}
	objs, err := s.s3.ListObjects(bucket, key)
	if err != nil {
		return nil, fmt.Errorf("%s can't be read: %v", p, err)
	}
	var keys []string
	for _, o := range objs {
		if !strings.HasSuffix(o.Key, "/") {
			keys = append(keys, o.Key)
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return nil, fmt.Errorf("%s has no objects", p)
	}
	var files []pszS3File
	total := 0
	for _, k := range keys {
		if !s.authorizeServiceS3As(c, principal, bucket, k, "s3:GetObject") {
			return nil, fmt.Errorf("access denied reading s3://%s/%s", bucket, k)
		}
		obj, err := s.s3.GetObject(bucket, k)
		if err != nil {
			return nil, fmt.Errorf("s3://%s/%s can't be read", bucket, k)
		}
		if total += len(obj.Body); total > pszMaxInputBytes {
			return nil, fmt.Errorf("%s is larger than the %d MiB Stackyard reads", p, pszMaxInputBytes>>20)
		}
		files = append(files, pszS3File{k, obj.Body})
	}
	return files, nil
}

// pszOutputBucket reports why output can't go to p, or "".
func (s *Server) pszOutputProblem(p string) string {
	bucket, _, err := pszS3Path(p)
	if err != nil {
		return err.Error()
	}
	if _, err := s.s3.GetBucket(bucket); err != nil {
		return fmt.Sprintf("Output bucket %s doesn't exist.", bucket)
	}
	return ""
}

// pszWriteS3 writes name under the prefix p as service acting for c, and
// returns why the write was denied, or "".
func (s *Server) pszWriteS3(c s3Caller, service, p, name string, body []byte) string {
	bucket, key, err := pszS3Path(p)
	if err != nil {
		return ""
	}
	if key != "" && !strings.HasSuffix(key, "/") {
		key += "/"
	}
	if !s.authorizeServiceS3As(c, service, bucket, key+name, "s3:PutObject") {
		return fmt.Sprintf("access denied writing s3://%s/%s", bucket, key+name)
	}
	_, _ = s.s3.PutObject(bucket, key+name, "application/octet-stream", body, nil, "", "")
	return ""
}

// failJob fails a job whose output couldn't be written after it was added.
func (st *personalizeStore) failJob(arn, reason string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if r := st.res[arn]; r != nil {
		r.next, r.failure = []string{pszStatusFailed}, reason
	}
}

// addJob registers a job; failure makes it settle to failedStatus.
func (st *personalizeStore) addJob(kind, arn string, fields map[string]any, rel map[string]string, tags [][2]string, failure string) (*pszRes, *pszErr) {
	if err := st.exists(arn); err != nil {
		return nil, err
	}
	r, err := st.add(kind, arn, fields, rel, tags, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if failure != "" {
		r.next, r.failure = []string{pszStatusFailed}, failure
	}
	return r, nil
}

func pszJobFields(in map[string]any, keys ...string) map[string]any {
	out := map[string]any{"jobName": pszStr(in, "jobName")}
	for _, k := range keys {
		if v, ok := in[k]; ok {
			out[k] = v
		}
	}
	return out
}

func (s *Server) pszCreateImportJob(in map[string]any) (map[string]any, *pszErr) {
	files, readErr := s.pszReadS3(s3CallerIn(in), pszStr(pszMap(in, "dataSource"), "dataLocation"))
	st := s.personalize
	st.mu.Lock()
	defer st.mu.Unlock()
	ds, err := st.active("dataset", pszStr(in, "datasetArn"))
	if err != nil {
		return nil, err
	}
	mode := pszStr(in, "importMode")
	if mode == "" {
		mode = "FULL"
	}
	failure := ""
	if readErr != nil {
		failure = readErr.Error()
	}
	merged := &pszImport{entities: map[string]map[string]string{}}
	for _, f := range files {
		imp, perr := pszParseCSV(f.body, ds.data)
		if perr != nil {
			failure = fmt.Sprintf("s3://.../%s: %v", path.Base(f.key), perr)
			break
		}
		for id, props := range imp.entities {
			merged.entities[id] = props
		}
		merged.events = append(merged.events, imp.events...)
	}
	fields := pszJobFields(in, "datasetArn", "dataSource", "roleArn", "publishAttributionMetricsToS3")
	fields["importMode"] = mode
	arn := pszARNBase + "dataset-import-job/" + pszStr(in, "jobName")
	if failure == "" && len(merged.entities)+len(ds.data.entities) > pszMaxRecords && mode == "INCREMENTAL" {
		failure = fmt.Sprintf("the dataset would exceed %d records", pszMaxRecords)
	}
	if _, err := st.addJob("importJob", arn, fields, map[string]string{"datasetArn": ds.arn}, pszTags(in), failure); err != nil {
		return nil, err
	}
	if failure == "" {
		if err := ds.data.apply(merged, mode == "FULL"); err != nil {
			st.res[arn].next, st.res[arn].failure = []string{pszStatusFailed}, err.msg
		}
	}
	return map[string]any{"datasetImportJobArn": arn}, nil
}

func (s *Server) pszCreateExportJob(in map[string]any) (map[string]any, *pszErr) {
	dest := pszStr(pszMap(pszMap(in, "jobOutput"), "s3DataDestination"), "path")
	failure := s.pszOutputProblem(dest)
	body, arn, perr := func() ([]byte, string, *pszErr) {
		st := s.personalize
		st.mu.Lock()
		defer st.mu.Unlock()
		ds, err := st.active("dataset", pszStr(in, "datasetArn"))
		if err != nil {
			return nil, "", err
		}
		mode := pszStr(in, "ingestionMode")
		if mode == "" {
			mode = "PUT"
		}
		fields := pszJobFields(in, "datasetArn", "roleArn", "jobOutput")
		fields["ingestionMode"] = mode
		arn := pszARNBase + "dataset-export-job/" + pszStr(in, "jobName")
		if _, err := st.addJob("exportJob", arn, fields, map[string]string{"datasetArn": ds.arn}, pszTags(in), failure); err != nil {
			return nil, "", err
		}
		return ds.data.export(mode), arn, nil
	}()
	if perr != nil {
		return nil, perr
	}
	if failure == "" {
		if denied := s.pszWriteS3(s3CallerIn(in), pszServicePrincipal, dest, pszStr(in, "jobName")+".csv", body); denied != "" {
			s.personalize.failJob(arn, denied)
		}
	}
	return map[string]any{"datasetExportJobArn": arn}, nil
}

func (s *Server) pszCreateDeletionJob(in map[string]any) (map[string]any, *pszErr) {
	files, readErr := s.pszReadS3(s3CallerIn(in), pszStr(pszMap(in, "dataSource"), "dataLocation"))
	st := s.personalize
	st.mu.Lock()
	defer st.mu.Unlock()
	g, err := st.active("datasetGroup", pszStr(in, "datasetGroupArn"))
	if err != nil {
		return nil, err
	}
	users := map[string]bool{}
	failure := ""
	if readErr != nil {
		failure = readErr.Error()
	}
	for _, f := range files {
		imp, perr := pszParseCSV(f.body, newPSZData(pszUsers, []string{"USER_ID"}))
		if perr != nil {
			failure = perr.Error()
			break
		}
		for id := range imp.entities {
			users[id] = true
		}
	}
	arn := pszARNBase + "data-deletion-job/" + pszStr(in, "jobName")
	r, err := st.addJob("deletionJob", arn, pszJobFields(in, "datasetGroupArn", "dataSource", "roleArn"),
		map[string]string{"datasetGroupArn": g.arn}, pszTags(in), failure)
	if err != nil {
		return nil, err
	}
	r.status = "IN_PROGRESS"
	if failure != "" {
		r.next = []string{"FAILED"}
		return map[string]any{"dataDeletionJobArn": arn}, nil
	}
	r.next = []string{"COMPLETED"}
	found := map[string]bool{}
	for _, typ := range []string{pszUsers, pszInteractions, pszActionInteractions} {
		d := st.dataset(g.arn, typ)
		if d == nil {
			continue
		}
		for id := range users {
			if d.data.entities[id] != nil {
				delete(d.data.entities, id)
				found[id] = true
			}
		}
		kept := d.data.events[:0:0]
		for _, e := range d.data.events {
			if users[e.user] {
				found[e.user] = true
			} else {
				kept = append(kept, e)
			}
		}
		d.data.events = kept
	}
	r.numDeleted = len(found)
	return map[string]any{"dataDeletionJobArn": arn}, nil
}

// batchPrep holds what a batch job needs from its solution version.
func (st *personalizeStore) batchVersion(in map[string]any) (*pszRes, *pszFilter, *pszErr) {
	v, err := st.active("solutionVersion", pszStr(in, "solutionVersionArn"))
	if err != nil {
		return nil, nil, err
	}
	f, err := st.filterFor(pszStr(in, "filterArn"), v.rel["datasetGroupArn"])
	return v, f, err
}

func pszLines(files []pszS3File, each func(line map[string]any) map[string]any) ([]pszS3File, error) {
	var out []pszS3File
	count, total := 0, 0
	for _, f := range files {
		var buf bytes.Buffer
		sc := bufio.NewScanner(bytes.NewReader(f.body))
		sc.Buffer(make([]byte, 64<<10), 1<<20)
		for sc.Scan() {
			text := strings.TrimSpace(sc.Text())
			if text == "" {
				continue
			}
			if count++; count > pszMaxBatchLines {
				return nil, fmt.Errorf("Stackyard processes at most %d input lines", pszMaxBatchLines)
			}
			var line map[string]any
			result := map[string]any{"input": text, "output": nil, "error": "the line isn't a JSON object"}
			if json.Unmarshal([]byte(text), &line) == nil {
				result = each(line)
				result["input"] = line
			}
			raw, _ := json.Marshal(result)
			if total += len(raw) + 1; total > pszMaxOutputBytes {
				return nil, fmt.Errorf("Stackyard writes at most %d bytes of batch output", pszMaxOutputBytes)
			}
			buf.Write(raw)
			buf.WriteByte('\n')
		}
		if err := sc.Err(); err != nil {
			return nil, err
		}
		out = append(out, pszS3File{path.Base(f.key) + ".out", buf.Bytes()})
	}
	return out, nil
}

func pszFilterValuesOf(line map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range pszMap(line, "filterValues") {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

func pszIDs(scored []pszScored) ([]any, []any) {
	ids, scores := make([]any, len(scored)), make([]any, len(scored))
	for i, s := range scored {
		ids[i], scores[i] = s.id, s.score
	}
	return ids, scores
}

func (s *Server) pszCreateBatchInferenceJob(in map[string]any) (map[string]any, *pszErr) {
	files, readErr := s.pszReadS3(s3CallerIn(in), pszStr(pszMap(pszMap(in, "jobInput"), "s3DataSource"), "path"))
	dest := pszStr(pszMap(pszMap(in, "jobOutput"), "s3DataDestination"), "path")
	outProblem := s.pszOutputProblem(dest)
	outputs, arn, perr := func() ([]pszS3File, string, *pszErr) {
		st := s.personalize
		st.mu.Lock()
		defer st.mu.Unlock()
		v, f, err := st.batchVersion(in)
		if err != nil {
			return nil, "", err
		}
		mode := pszStr(in, "batchInferenceJobMode")
		if mode == "" {
			mode = "BATCH_INFERENCE"
		}
		rt := v.recipe.recipeType
		switch {
		case rt == pszUserSegmentation:
			return nil, "", pszInvalid("%s is a USER_SEGMENTATION recipe; use CreateBatchSegmentJob.", v.recipe.name)
		case mode == "THEME_GENERATION" && (rt != pszRelatedItems || pszMap(in, "themeGenerationConfig") == nil):
			return nil, "", pszInvalid("THEME_GENERATION needs a RELATED_ITEMS solution version and a themeGenerationConfig.")
		}
		n := pszInt(in, "numResults", 25)
		if n < 1 || n > pszMaxRecommendations {
			return nil, "", pszInvalid("numResults must be between 1 and %d.", pszMaxRecommendations)
		}
		g := st.group(v.rel["datasetGroupArn"])
		outputs, lineErr := pszLines(files, func(line map[string]any) map[string]any {
			values := pszFilterValuesOf(line)
			var scored []pszScored
			var perr *pszErr
			key := "recommendedItems"
			switch rt {
			case pszPersonalizedRanking:
				scored, perr = g.rank(v.arn, pszStr(line, "userId"), pszStrings(line["itemList"]), f, values)
			case pszPersonalizedActions:
				key = "recommendedActions"
				scored, perr = g.recommendActions(v.arn, pszStr(line, "userId"), n, f, values)
			default:
				scored, perr = g.recommendItems(v.arn, rt, pszStr(line, "userId"), pszStr(line, "itemId"), n, f, values)
			}
			if perr != nil {
				return map[string]any{"output": nil, "error": perr.msg}
			}
			ids, scores := pszIDs(scored)
			out := map[string]any{key: ids, "scores": scores}
			if mode == "THEME_GENERATION" {
				out["theme"] = "Items like " + pszStr(line, "itemId")
			}
			return map[string]any{"output": out, "error": nil}
		})
		failure := outProblem
		if readErr != nil {
			failure = readErr.Error()
		} else if lineErr != nil {
			failure = lineErr.Error()
		}
		fields := pszJobFields(in, "solutionVersionArn", "filterArn", "jobInput", "jobOutput", "roleArn", "batchInferenceJobConfig", "themeGenerationConfig")
		fields["numResults"], fields["batchInferenceJobMode"] = n, mode
		arn := pszARNBase + "batch-inference-job/" + pszStr(in, "jobName")
		if _, err := st.addJob("batchInferenceJob", arn, fields, map[string]string{"solutionVersionArn": v.arn}, pszTags(in), failure); err != nil {
			return nil, "", err
		}
		if failure != "" {
			outputs = nil
		}
		return outputs, arn, nil
	}()
	if perr != nil {
		return nil, perr
	}
	for _, o := range outputs {
		if denied := s.pszWriteS3(s3CallerIn(in), pszServicePrincipal, dest, o.key, o.body); denied != "" {
			s.personalize.failJob(arn, denied)
			break
		}
	}
	return map[string]any{"batchInferenceJobArn": arn}, nil
}

func (s *Server) pszCreateBatchSegmentJob(in map[string]any) (map[string]any, *pszErr) {
	files, readErr := s.pszReadS3(s3CallerIn(in), pszStr(pszMap(pszMap(in, "jobInput"), "s3DataSource"), "path"))
	dest := pszStr(pszMap(pszMap(in, "jobOutput"), "s3DataDestination"), "path")
	outProblem := s.pszOutputProblem(dest)
	outputs, arn, perr := func() ([]pszS3File, string, *pszErr) {
		st := s.personalize
		st.mu.Lock()
		defer st.mu.Unlock()
		v, f, err := st.batchVersion(in)
		if err != nil {
			return nil, "", err
		}
		if v.recipe.recipeType != pszUserSegmentation {
			return nil, "", pszInvalid("Batch segment jobs need a USER_SEGMENTATION solution version; %s is %s.", v.recipe.name, v.recipe.recipeType)
		}
		n := pszInt(in, "numResults", 25)
		if n < 1 || n > pszMaxRecords {
			return nil, "", pszInvalid("numResults must be between 1 and %d.", pszMaxRecords)
		}
		g := st.group(v.rel["datasetGroupArn"])
		outputs, lineErr := pszLines(files, func(line map[string]any) map[string]any {
			items := map[string]bool{}
			if id := pszStr(line, "itemId"); id != "" {
				items[id] = true
			} else if expr := pszStr(line, "itemAttributes"); expr != "" {
				attr, perr := pszParseFilter("INCLUDE ItemID WHERE " + expr)
				if perr != nil {
					return map[string]any{"output": nil, "error": perr.msg}
				}
				ev := newPSZEval(g, attr, "", nil)
				for _, id := range g.items.entityIDs() {
					if ok, _ := ev.allows(id); ok {
						items[id] = true
					}
				}
			} else {
				return map[string]any{"output": nil, "error": "each line needs itemId or itemAttributes"}
			}
			scored, perr := g.segment(v.arn, items, n, f, pszFilterValuesOf(line))
			if perr != nil {
				return map[string]any{"output": nil, "error": perr.msg}
			}
			ids, _ := pszIDs(scored)
			return map[string]any{"output": map[string]any{"usersList": ids}, "error": nil}
		})
		failure := outProblem
		if readErr != nil {
			failure = readErr.Error()
		} else if lineErr != nil {
			failure = lineErr.Error()
		}
		fields := pszJobFields(in, "solutionVersionArn", "filterArn", "jobInput", "jobOutput", "roleArn")
		fields["numResults"] = n
		arn := pszARNBase + "batch-segment-job/" + pszStr(in, "jobName")
		if _, err := st.addJob("batchSegmentJob", arn, fields, map[string]string{"solutionVersionArn": v.arn}, pszTags(in), failure); err != nil {
			return nil, "", err
		}
		if failure != "" {
			outputs = nil
		}
		return outputs, arn, nil
	}()
	if perr != nil {
		return nil, perr
	}
	for _, o := range outputs {
		if denied := s.pszWriteS3(s3CallerIn(in), pszServicePrincipal, dest, o.key, o.body); denied != "" {
			s.personalize.failJob(arn, denied)
			break
		}
	}
	return map[string]any{"batchSegmentJobArn": arn}, nil
}
