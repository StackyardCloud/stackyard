package server

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Earth observation and vector enrichment jobs. Jobs go INITIALIZING ->
// IN_PROGRESS -> COMPLETED (FAILED when their S3 input is unusable) and
// STOPPING -> STOPPED, one step per read; exports go IN_PROGRESS -> SUCCEEDED and write real objects to
// Stackyard's S3. S3 is touched only outside the store lock.

const (
	sgeoMaxJobs  = 200 // per job type
	sgeoMaxTags  = 50
	sgeoMaxBytes = 32 << 20
	sgeoMaxZoom  = 24
)

var sgeoOperationTypes = map[string]string{
	"BandMathConfig": "BAND_MATH", "CloudMaskingConfig": "CLOUD_MASKING", "CloudRemovalConfig": "CLOUD_REMOVAL",
	"GeoMosaicConfig": "GEOMOSAIC", "LandCoverSegmentationConfig": "LAND_COVER_SEGMENTATION", "ResamplingConfig": "RESAMPLING",
	"StackConfig": "STACK", "TemporalStatisticsConfig": "TEMPORAL_STATISTICS", "ZonalStatisticsConfig": "ZONAL_STATISTICS",
}

type sgeoJob struct {
	kind       string // "earth-observation-job" or "vector-enrichment-job"
	arn        string
	doc        map[string]any
	created    time.Time
	failure    string
	exportFail string
	token      string
	size       int
}

type sgeoStore struct {
	mu     sync.Mutex
	jobs   map[string]*sgeoJob
	tokens map[string]string // client token -> job ARN
	counts map[string]int
	bytes  int
}

func newSageMakerGeospatialStore() *sgeoStore {
	return &sgeoStore{jobs: map[string]*sgeoJob{}, tokens: map[string]string{}, counts: map[string]int{}}
}

// sgeoPre carries what was learned from (or will be written to) S3 outside the lock.
type sgeoPre struct {
	failure string
	writes  []sgeoWrite
	tiles   []string // export: one PNG per output band, rendered after the lock
	prefix  string   // export: where the tiles go
	bucket  string
}

type sgeoWrite struct {
	bucket, key, contentType string
	body                     []byte
}

func (s *Server) sgeoRun(op string, in map[string]any) (any, *fcErr) {
	pre := &sgeoPre{}
	switch op {
	case "StartEarthObservationJob":
		if zone := fcStr(fcObj(fcObj(in, "JobConfig"), "ZonalStatisticsConfig"), "ZoneS3Path"); zone != "" {
			if _, err := s.pszReadS3As(s3CallerIn(in), sgeoServicePrincipal, zone); err != nil {
				pre.failure = "ZoneS3Path " + zone + " can't be read: " + err.Error()
			}
		}
	case "StartVectorEnrichmentJob":
		pre.failure = s.sgeoCheckVectorInput(in)
	case "ExportEarthObservationJob", "ExportVectorEnrichmentJob":
		uri := fcStr(fcObj(fcObj(in, "OutputConfig"), "S3Data"), "S3Uri")
		if problem := s.pszOutputProblem(uri); problem != "" {
			pre.failure = problem
		} else if op == "ExportVectorEnrichmentJob" {
			s.sgeoPrepareVectorExport(in, pre)
		}
	}
	out, err := s.sageMakerGeospatial.run(op, in, pre)
	if err != nil {
		return nil, err
	}
	for i, band := range pre.tiles {
		if i >= 32 {
			break
		}
		pre.writes = append(pre.writes, sgeoWrite{pre.bucket, pre.prefix + "output/" + band + ".png", "image/png", sgeoTile(fcStr(in, "Arn")+band, 0, 0, 0)})
	}
	// Authorize every object before writing any; a denied export writes
	// nothing and reports FAILED.
	for _, w := range pre.writes {
		if !s.authorizeServiceS3As(s3CallerIn(in), sgeoServicePrincipal, w.bucket, w.key, "s3:PutObject") {
			s.sageMakerGeospatial.failExport(fcStr(in, "Arn"), fmt.Sprintf("access denied writing s3://%s/%s", w.bucket, w.key))
			return out, nil
		}
	}
	for _, w := range pre.writes {
		_, _ = s.s3.PutObject(w.bucket, w.key, w.contentType, w.body, nil, "", "")
	}
	return out, nil
}

func (st *sgeoStore) run(op string, in map[string]any, pre *sgeoPre) (any, *fcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	out, err := st.dispatch(op, in, pre)
	if err != nil {
		pre.writes, pre.tiles = nil, nil // nothing reaches S3 for a refused request
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	m := awsmodels.SageMakerGeospatialModel()
	return m.Conform(m.Operations[op].Output, out), nil
}

func (st *sgeoStore) dispatch(op string, in map[string]any, pre *sgeoPre) (map[string]any, *fcErr) {
	const eo, ve = "earth-observation-job", "vector-enrichment-job"
	switch op {
	case "StartEarthObservationJob":
		return st.startEOJ(in, pre)
	case "StartVectorEnrichmentJob":
		return st.startVEJ(in, pre)
	case "GetEarthObservationJob":
		return st.getDoc(eo, fcStr(in, "Arn"))
	case "GetVectorEnrichmentJob":
		return st.getDoc(ve, fcStr(in, "Arn"))
	case "ListEarthObservationJobs":
		return st.list(eo, in)
	case "ListVectorEnrichmentJobs":
		return st.list(ve, in)
	case "StopEarthObservationJob":
		return nil, st.stop(eo, fcStr(in, "Arn"))
	case "StopVectorEnrichmentJob":
		return nil, st.stop(ve, fcStr(in, "Arn"))
	case "DeleteEarthObservationJob":
		return nil, st.delete(eo, fcStr(in, "Arn"))
	case "DeleteVectorEnrichmentJob":
		return nil, st.delete(ve, fcStr(in, "Arn"))
	case "ExportEarthObservationJob":
		return st.export(eo, in, pre)
	case "ExportVectorEnrichmentJob":
		return st.export(ve, in, pre)
	case "GetRasterDataCollection":
		return st.getCollection(in)
	case "ListRasterDataCollections":
		return st.listCollections(in)
	case "SearchRasterDataCollection":
		return st.search(in)
	case "TagResource", "UntagResource", "ListTagsForResource":
		return st.tagOp(op, in)
	}
	return nil, fcError("UnknownOperationException", "unknown operation %s", op)
}

func sgeoNewArn(kind string) string {
	return "arn:aws:sagemaker-geospatial:" + sgeoRegion + ":" + fcAccount + ":" + kind + "/" + lkeHex(8)
}

func sgeoISO(v any) string {
	return time.Unix(int64(amlTime(v)), 0).UTC().Format(time.RFC3339)
}

// job returns a job of kind, advancing its status and export.
// failExport marks an export that couldn't write its output as failed; the
// next read settles it FAILED with reason.
func (st *sgeoStore) failExport(arn, reason string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if j := st.jobs[arn]; j != nil {
		j.exportFail = reason
	}
}

func (st *sgeoStore) job(kind, arn string) (*sgeoJob, *fcErr) {
	j := st.jobs[arn]
	if j == nil || j.kind != kind {
		return nil, fcNotFound("%s %s does not exist", kind, arn)
	}
	st.advance(j)
	return j, nil
}

// advance moves j one status step per read and finishes an export on the
// read after it began. ponytail: reads, not wall time, drive progress, so a
// job just started can always be stopped however slow the caller is.
func (st *sgeoStore) advance(j *sgeoJob) {
	now := time.Now().UTC()
	if next := map[any]string{"INITIALIZING": "IN_PROGRESS", "IN_PROGRESS": "COMPLETED", "STOPPING": "STOPPED"}[j.doc["Status"]]; next != "" {
		if next == "COMPLETED" && j.failure != "" {
			next = "FAILED"
			if j.kind == "vector-enrichment-job" {
				j.doc["ErrorDetails"] = map[string]any{"ErrorType": "CLIENT_ERROR", "ErrorMessage": j.failure}
			} else {
				j.doc["ErrorDetails"] = map[string]any{"Type": "CLIENT_ERROR", "Message": j.failure}
			}
		}
		j.doc["Status"] = next
		if next != "IN_PROGRESS" {
			j.doc["DurationInSeconds"] = int(now.Sub(j.created).Seconds())
		}
	}
	if j.doc["ExportStatus"] != "IN_PROGRESS" {
		return
	}
	if j.exportFail == "" {
		j.doc["ExportStatus"] = "SUCCEEDED"
		return
	}
	j.doc["ExportStatus"] = "FAILED"
	detail := map[string]any{"Type": "CLIENT_ERROR", "Message": j.exportFail}
	if j.kind == "earth-observation-job" {
		detail = map[string]any{"ExportResults": detail}
	}
	j.doc["ExportErrorDetails"] = detail
}

func sgeoBusy(j *sgeoJob) bool {
	s := j.doc["Status"]
	return s == "INITIALIZING" || s == "IN_PROGRESS" || s == "STOPPING" || j.doc["ExportStatus"] == "IN_PROGRESS"
}

// admit makes room for a job of kind, evicting the oldest finished one when full.
func (st *sgeoStore) admit(kind string) *fcErr {
	if st.counts[kind] < sgeoMaxJobs {
		return nil
	}
	var oldest *sgeoJob
	for _, j := range st.jobs {
		if j.kind == kind && !sgeoBusy(j) && (oldest == nil || j.created.Before(oldest.created)) {
			oldest = j
		}
	}
	if oldest == nil {
		return fcError("ServiceQuotaExceededException", "at most %d %s resources", sgeoMaxJobs, kind)
	}
	st.remove(oldest)
	return nil
}

func (st *sgeoStore) remove(j *sgeoJob) {
	delete(st.jobs, j.arn)
	if j.token != "" && st.tokens[j.token] == j.arn {
		delete(st.tokens, j.token)
	}
	st.counts[j.kind]--
	st.bytes -= j.size
}

// charge re-sizes j against the byte cap.
func (st *sgeoStore) charge(j *sgeoJob) *fcErr {
	size := amlSize(j.doc)
	if st.bytes-j.size+size > sgeoMaxBytes {
		return fcError("ServiceQuotaExceededException", "Stackyard holds at most %d MiB of SageMaker geospatial jobs", sgeoMaxBytes>>20)
	}
	st.bytes += size - j.size
	j.size = size
	return nil
}

// add stores a new job unless its client token was already used.
func (st *sgeoStore) add(kind string, in, doc map[string]any, failure string) (map[string]any, *fcErr) {
	token := fcStr(in, "ClientToken")
	if arn, ok := st.tokens[token]; ok && token != "" {
		if j := st.jobs[arn]; j != nil && j.kind == kind {
			return j.doc, nil
		}
	}
	if err := st.admit(kind); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	arn := sgeoNewArn(kind)
	doc["Arn"], doc["CreationTime"], doc["DurationInSeconds"], doc["Status"] = arn, fcEpoch(now), 0, "INITIALIZING"
	doc["Name"], doc["ExecutionRoleArn"] = fcStr(in, "Name"), fcStr(in, "ExecutionRoleArn")
	if v := fcStr(in, "KmsKeyId"); v != "" {
		doc["KmsKeyId"] = v
	}
	tags := fcObj(in, "Tags")
	if len(tags) > sgeoMaxTags {
		return nil, sgeoInvalid("a job can have at most %d tags", sgeoMaxTags)
	}
	doc["Tags"] = tags
	j := &sgeoJob{kind: kind, arn: arn, doc: doc, created: now, failure: failure, token: token}
	if err := st.charge(j); err != nil {
		return nil, err
	}
	st.jobs[arn] = j
	st.counts[kind]++
	if token != "" {
		st.tokens[token] = arn
	}
	return doc, nil
}

// sgeoOneOf returns the single member set on a union, or an error.
func sgeoOneOf(name string, union map[string]any) (string, *fcErr) {
	if len(union) != 1 {
		return "", sgeoInvalid("%s must set exactly one member", name)
	}
	for k := range union {
		return k, nil
	}
	return "", nil
}

func sgeoNames(bands []any) []string {
	out := make([]string, 0, len(bands))
	for _, b := range bands {
		out = append(out, fcStr(asMap(b), "BandName"))
	}
	return out
}

// sgeoOutputBands derives a job's output bands from its operation and input bands.
func sgeoOutputBands(op string, cfg map[string]any, input []string) ([]any, *fcErr) {
	targets := fcStrs(cfg, "TargetBands")
	for _, b := range targets {
		if !fcContainsAny(fcAnyList(input), b) {
			return nil, sgeoInvalid("target band %s is not an input band", b)
		}
	}
	if len(targets) == 0 {
		targets = input
	}
	band := func(name, typ string) any { return map[string]any{"BandName": name, "OutputDataType": typ} }
	var out []any
	switch op {
	case "BandMathConfig":
		for _, name := range fcStrs(cfg, "PredefinedIndices") {
			out = append(out, band(name, "FLOAT32"))
		}
		ops, _ := fcObj(cfg, "CustomIndices")["Operations"].([]any)
		for _, o := range ops {
			typ := fcStr(asMap(o), "OutputType")
			if typ == "" {
				typ = "FLOAT32"
			}
			out = append(out, band(fcStr(asMap(o), "Name"), typ))
		}
		if len(out) == 0 {
			return nil, sgeoInvalid("BandMathConfig needs PredefinedIndices or CustomIndices")
		}
	case "LandCoverSegmentationConfig":
		out = append(out, band("mask", "INT32"))
	case "CloudMaskingConfig":
		for _, b := range targets {
			out = append(out, band(b, "UINT16"))
		}
		out = append(out, band("mask", "INT32"))
	case "TemporalStatisticsConfig", "ZonalStatisticsConfig":
		for _, b := range targets {
			for _, stat := range fcStrs(cfg, "Statistics") {
				out = append(out, band(b+"_"+strings.ToLower(stat), "FLOAT32"))
			}
		}
	default:
		for _, b := range targets {
			out = append(out, band(b, "UINT16"))
		}
	}
	return out, nil
}

func (st *sgeoStore) startEOJ(in map[string]any, pre *sgeoPre) (map[string]any, *fcErr) {
	input := fcObj(in, "InputConfig")
	src, err := sgeoOneOf("InputConfig", input)
	if err != nil {
		return nil, err
	}
	outInput := map[string]any{}
	var bands []string
	if src == "PreviousEarthObservationJobArn" {
		prev, err := st.job("earth-observation-job", fcStr(input, src))
		if err != nil {
			return nil, err
		}
		if prev.doc["Status"] != "COMPLETED" {
			return nil, sgeoConflict("previous job %s is %s, not COMPLETED", prev.arn, prev.doc["Status"])
		}
		outInput[src], bands = prev.arn, sgeoNames(asList(prev.doc["OutputBands"]))
	} else {
		q := fcObj(input, "RasterDataCollectionQuery")
		c := sgeoCollectionByArn(fcStr(q, "RasterDataCollectionArn"))
		if c == nil {
			return nil, fcNotFound("Raster data collection %s does not exist", fcStr(q, "RasterDataCollectionArn"))
		}
		tr := fcObj(q, "TimeRangeFilter")
		if amlTime(tr["EndTime"]) <= amlTime(tr["StartTime"]) {
			return nil, sgeoInvalid("TimeRangeFilter EndTime must be after StartTime")
		}
		outQ := lkePick(q, "AreaOfInterest", "PropertyFilters", "RasterDataCollectionArn")
		outQ["RasterDataCollectionName"] = c.name
		outQ["TimeRangeFilter"] = map[string]any{"StartTime": sgeoISO(tr["StartTime"]), "EndTime": sgeoISO(tr["EndTime"])}
		outInput["RasterDataCollectionQuery"], bands = outQ, c.bands
	}
	cfg := fcObj(in, "JobConfig")
	op, err := sgeoOneOf("JobConfig", cfg)
	if err != nil {
		return nil, err
	}
	outBands, err := sgeoOutputBands(op, fcObj(cfg, op), bands)
	if err != nil {
		return nil, err
	}
	doc := map[string]any{"InputConfig": outInput, "JobConfig": cfg, "OutputBands": outBands, "_op": sgeoOperationTypes[op]}
	return st.add("earth-observation-job", in, doc, pre.failure)
}

func asList(v any) []any {
	l, _ := v.([]any)
	return l
}

func (st *sgeoStore) startVEJ(in map[string]any, pre *sgeoPre) (map[string]any, *fcErr) {
	cfg := fcObj(in, "JobConfig")
	op, err := sgeoOneOf("JobConfig", cfg)
	if err != nil {
		return nil, err
	}
	if _, err := sgeoOneOf("DataSourceConfig", fcObj(fcObj(in, "InputConfig"), "DataSourceConfig")); err != nil {
		return nil, err
	}
	typ := map[string]string{"ReverseGeocodingConfig": "REVERSE_GEOCODING", "MapMatchingConfig": "MAP_MATCHING"}[op]
	doc := map[string]any{"InputConfig": fcObj(in, "InputConfig"), "JobConfig": cfg, "Type": typ}
	return st.add("vector-enrichment-job", in, doc, pre.failure)
}

func (st *sgeoStore) getDoc(kind, arn string) (map[string]any, *fcErr) {
	j, err := st.job(kind, arn)
	if err != nil {
		return nil, err
	}
	return j.doc, nil
}

func (st *sgeoStore) list(kind string, in map[string]any) (map[string]any, *fcErr) {
	status := fcStr(in, "StatusEquals")
	var items []map[string]any
	for _, j := range st.jobs {
		if j.kind != kind {
			continue
		}
		st.advance(j)
		if status != "" && j.doc["Status"] != status {
			continue
		}
		item := lkePick(j.doc, "Arn", "CreationTime", "DurationInSeconds", "Name", "Status", "Tags", "Type")
		item["OperationType"] = j.doc["_op"]
		items = append(items, item)
	}
	sortBy := fcStr(in, "SortBy")
	if sortBy != "Name" {
		sortBy = "CreationTime"
	}
	desc := fcStr(in, "SortOrder") == "DESCENDING"
	sort.Slice(items, func(a, b int) bool {
		c := amlCompare(map[string]string{"CreationTime": "CreatedAt"}[sortBy], items[a][sortBy], items[b][sortBy])
		if c == 0 {
			c = strings.Compare(fcStr(items[a], "Arn"), fcStr(items[b], "Arn"))
		}
		return (c < 0) != desc
	})
	listKey := map[string]string{"earth-observation-job": "EarthObservationJobSummaries", "vector-enrichment-job": "VectorEnrichmentJobSummaries"}[kind]
	return sgeoPage(items, in, listKey)
}

func (st *sgeoStore) stop(kind, arn string) *fcErr {
	j, err := st.job(kind, arn)
	if err != nil {
		return err
	}
	if s := j.doc["Status"]; s != "INITIALIZING" && s != "IN_PROGRESS" {
		return sgeoConflict("job %s is %s and can't be stopped", arn, s)
	}
	j.doc["Status"] = "STOPPING"
	return nil
}

func (st *sgeoStore) delete(kind, arn string) *fcErr {
	j, err := st.job(kind, arn)
	if err != nil {
		return err
	}
	if sgeoBusy(j) {
		return sgeoConflict("job %s is %s; stop it (and let exports finish) before deleting it", arn, j.doc["Status"])
	}
	st.remove(j)
	return nil
}

// export marks the job exporting and fills pre with the objects to write.
func (st *sgeoStore) export(kind string, in map[string]any, pre *sgeoPre) (map[string]any, *fcErr) {
	j, err := st.job(kind, fcStr(in, "Arn"))
	if err != nil {
		return nil, err
	}
	if j.doc["Status"] != "COMPLETED" {
		return nil, sgeoConflict("job %s is %s; only COMPLETED jobs export", j.arn, j.doc["Status"])
	}
	if j.doc["ExportStatus"] == "IN_PROGRESS" {
		return nil, sgeoConflict("job %s is already exporting", j.arn)
	}
	doc := lkeCopy(j.doc)
	out := fcObj(in, "OutputConfig")
	doc["ExportStatus"], doc["OutputConfig"] = "IN_PROGRESS", out
	delete(doc, "ExportErrorDetails")
	if kind == "earth-observation-job" {
		doc["ExportSourceImages"] = fcBool(in, "ExportSourceImages")
	}
	old := j.doc
	j.doc = doc
	if err := st.charge(j); err != nil {
		j.doc = old
		return nil, err
	}
	j.exportFail = pre.failure
	if pre.failure == "" && kind == "earth-observation-job" {
		bucket, prefix, _ := pszS3Path(fcStr(fcObj(out, "S3Data"), "S3Uri"))
		if prefix != "" && !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		prefix += strings.TrimPrefix(j.arn[strings.LastIndex(j.arn, "/"):], "/") + "/"
		manifest, _ := json.Marshal(map[string]any{"Arn": j.arn, "Name": j.doc["Name"], "OutputBands": j.doc["OutputBands"],
			"InputConfig": j.doc["InputConfig"], "JobConfig": j.doc["JobConfig"]})
		pre.bucket, pre.prefix, pre.tiles = bucket, prefix, sgeoNames(asList(j.doc["OutputBands"]))
		pre.writes = append(pre.writes, sgeoWrite{bucket, prefix + "manifest.json", "application/json", manifest})
		if fcBool(in, "ExportSourceImages") {
			src, _ := json.Marshal(map[string]any{"InputConfig": j.doc["InputConfig"]})
			pre.writes = append(pre.writes, sgeoWrite{bucket, prefix + "source/manifest.json", "application/json", src})
		}
	}
	res := lkePick(doc, "Arn", "CreationTime", "ExecutionRoleArn", "ExportStatus", "OutputConfig", "ExportSourceImages")
	res["ExecutionRoleArn"] = fcStr(in, "ExecutionRoleArn")
	return res, nil
}

func (st *sgeoStore) tagOp(op string, in map[string]any) (map[string]any, *fcErr) {
	arn := fcStr(in, "ResourceArn")
	j := st.jobs[arn]
	if j == nil {
		if c := sgeoCollectionByArn(arn); c != nil && op == "ListTagsForResource" {
			return map[string]any{"Tags": map[string]any{}}, nil
		}
		return nil, fcNotFound("Resource %s does not exist", arn)
	}
	tags := lkeCopy(fcObj(j.doc, "Tags"))
	switch op {
	case "ListTagsForResource":
		return map[string]any{"Tags": tags}, nil
	case "TagResource":
		for k, v := range fcObj(in, "Tags") {
			tags[k] = v
		}
		if len(tags) > sgeoMaxTags {
			return nil, sgeoInvalid("a job can have at most %d tags", sgeoMaxTags)
		}
	default:
		for _, k := range fcStrs(in, "TagKeys") {
			delete(tags, k)
		}
	}
	doc := lkeCopy(j.doc)
	doc["Tags"] = tags
	old := j.doc
	j.doc = doc
	if err := st.charge(j); err != nil {
		j.doc = old
		return nil, err
	}
	return nil, nil
}

// tileSeed checks a GetTile request and returns what the tile is rendered from.
func (st *sgeoStore) tileSeed(in map[string]any) (string, int, int, int, *fcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	j, err := st.job("earth-observation-job", fcStr(in, "Arn"))
	if err != nil {
		return "", 0, 0, 0, err
	}
	xf, _ := fcNum(in, "x")
	yf, _ := fcNum(in, "y")
	zf, _ := fcNum(in, "z")
	if zf < 0 || zf > sgeoMaxZoom {
		return "", 0, 0, 0, sgeoInvalid("z must be between 0 and %d", sgeoMaxZoom)
	}
	x, y, z := int(xf), int(yf), int(zf)
	if n := 1 << z; x < 0 || y < 0 || x >= n || y >= n {
		return "", 0, 0, 0, sgeoInvalid("tile %d/%d/%d is outside the zoom level's grid", z, x, y)
	}
	target := fcStr(in, "Target")
	var bands []string
	if target == "OUTPUT" {
		if j.doc["Status"] != "COMPLETED" {
			return "", 0, 0, 0, sgeoConflict("job %s is %s; output tiles exist once it is COMPLETED", j.arn, j.doc["Status"])
		}
		bands = sgeoNames(asList(j.doc["OutputBands"]))
	} else if c := sgeoCollectionByArn(fcStr(fcObj(fcObj(j.doc, "InputConfig"), "RasterDataCollectionQuery"), "RasterDataCollectionArn")); c != nil {
		bands = c.bands
	}
	assets := fcStrs(in, "ImageAssets")
	for _, a := range assets {
		if !fcContainsAny(fcAnyList(bands), a) {
			return "", 0, 0, 0, sgeoInvalid("image asset %s is not a %s band of job %s", a, strings.ToLower(target), j.arn)
		}
	}
	return fmt.Sprintf("%s|%s|%s", j.arn, target, strings.Join(assets, ",")), x, y, z, nil
}
