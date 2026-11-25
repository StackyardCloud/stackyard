package server

import (
	"strings"
)

// transfer runs an execution's transfer phase. S3-to-S3 tasks on plain bucket
// locations really copy objects between Stackyard buckets, honouring the
// filters and the overwrite, transfer and delete options; other location
// pairs are simulated as an empty, successful transfer.
func (st *dsyStore) transfer(s *Server, e *dsyExec) *dsyErr {
	t := st.tasks[e.task]
	if t == nil {
		return &dsyErr{code: "TaskDeleted", msg: "the task was deleted"}
	}
	src, dst := st.locations[t.src], st.locations[t.dst]
	srcBucket, srcPrefix, ok1 := dsyS3Root(src)
	dstBucket, dstPrefix, ok2 := dsyS3Root(dst)
	if !ok1 || !ok2 {
		// ponytail: only S3-to-S3 moves data; NFS/SMB/EFS/FSx/... have no
		// file content in Stackyard to copy.
		return nil
	}
	denied := func(action, bucket, key string) *dsyErr {
		if s.authorizeServiceS3As(e.caller, "datasync.amazonaws.com", bucket, key, action) {
			return nil
		}
		return &dsyErr{code: "AccessDenied", msg: "access denied: " + action + " on s3://" + bucket + "/" + key}
	}
	if err := denied("s3:ListBucket", srcBucket, ""); err != nil {
		return err
	}
	if err := denied("s3:ListBucket", dstBucket, ""); err != nil {
		return err
	}
	objects, err := s.s3.ListObjects(srcBucket, srcPrefix)
	if err != nil {
		return &dsyErr{code: "SourceUnavailable", msg: "source bucket " + srcBucket + " is unavailable"}
	}
	if len(objects) > dsyMaxCopyObjects {
		return &dsyErr{code: "TooManyObjects", msg: "Stackyard copies at most 10000 objects per execution"}
	}
	existing, err := s.s3.ListObjects(dstBucket, dstPrefix)
	if err != nil {
		return &dsyErr{code: "DestinationUnavailable", msg: "destination bucket " + dstBucket + " is unavailable"}
	}
	destByKey := make(map[string]int, len(existing))
	for i, o := range existing {
		destByKey[o.Key] = i
	}
	e.counts["FilesListed"], e.counts["DestListed"] = float64(len(objects)), float64(len(existing))

	var total int
	sourceRel := make(map[string]bool, len(objects))
	type copyJob struct{ srcKey, dstKey string }
	var jobs []copyJob
	for _, o := range objects {
		rel := strings.TrimPrefix(o.Key, srcPrefix)
		if !dsyIncluded("/"+rel, e.includes, e.excludes) {
			continue
		}
		sourceRel[rel] = true
		dstKey := dstPrefix + rel
		if i, exists := destByKey[dstKey]; exists {
			d := existing[i]
			if e.options["OverwriteMode"] == "NEVER" ||
				(e.options["TransferMode"] == "CHANGED" && d.ETag == o.ETag && d.SizeBytes == o.SizeBytes) {
				e.counts["FilesSkipped"]++
				continue
			}
		}
		total += o.SizeBytes
		if total > dsyMaxCopyBytes {
			return &dsyErr{code: "TooMuchData", msg: "Stackyard copies at most 256 MiB per execution"}
		}
		jobs = append(jobs, copyJob{o.Key, dstKey})
	}
	e.counts["EstimatedFilesToTransfer"], e.counts["EstimatedBytesToTransfer"], e.counts["FilesPrepared"] = float64(len(jobs)), float64(total), float64(len(jobs))

	storageClass := dsyStr(dst.attrs, "S3StorageClass")
	var metadata map[string]string
	if e.options["ObjectMetadata"] == "NONE" {
		metadata = map[string]string{}
	}
	for _, j := range jobs {
		if err := denied("s3:GetObject", srcBucket, j.srcKey); err != nil {
			return err
		}
		if err := denied("s3:PutObject", dstBucket, j.dstKey); err != nil {
			return err
		}
	}
	for _, j := range jobs {
		obj, err := s.s3.CopyObject(srcBucket, j.srcKey, dstBucket, j.dstKey, metadata, "", "", storageClass)
		if err != nil {
			return &dsyErr{code: "CopyFailed", msg: "copying " + j.srcKey + " failed: " + err.Error()}
		}
		e.counts["FilesTransferred"]++
		e.counts["BytesTransferred"] += float64(obj.SizeBytes)
		e.counts["BytesWritten"] += float64(obj.SizeBytes)
	}
	if e.options["PreserveDeletedFiles"] == "REMOVE" {
		for _, d := range existing {
			if rel := strings.TrimPrefix(d.Key, dstPrefix); !sourceRel[rel] && dsyIncluded("/"+rel, e.includes, e.excludes) {
				if err := denied("s3:DeleteObject", dstBucket, d.Key); err != nil {
					return err
				}
				if err := s.s3.DeleteObject(dstBucket, d.Key); err == nil {
					e.counts["FilesDeleted"]++
				}
			}
		}
		e.counts["EstimatedFilesToDelete"] = e.counts["FilesDeleted"]
	}
	return nil
}

// dsyS3Root returns the bucket and key prefix of a plain S3 bucket location.
func dsyS3Root(loc *dsyLocation) (bucket, prefix string, ok bool) {
	if loc == nil || loc.kind != "S3" {
		return "", "", false
	}
	m := dsyS3BucketARNPattern.FindStringSubmatch(dsyStr(loc.attrs, "S3BucketArn"))
	if m == nil {
		return "", "", false
	}
	return m[1], strings.TrimPrefix(dsySubdir(dsyStr(loc.attrs, "Subdirectory")), "/"), true
}

// dsyIncluded applies SIMPLE_PATTERN include and exclude filters to a path
// relative to the location root ("/dir/file").
func dsyIncluded(rel string, includes, excludes []any) bool {
	matchAny := func(filters []any) bool {
		for _, raw := range filters {
			f, _ := raw.(map[string]any)
			for _, p := range strings.Split(dsyStr(f, "Value"), "|") {
				if dsyPatternMatch(p, rel) {
					return true
				}
			}
		}
		return false
	}
	if len(includes) > 0 && !matchAny(includes) {
		return false
	}
	return !matchAny(excludes)
}

// dsyPatternMatch matches a pattern where * matches any run of characters
// (including /); a pattern without a wildcard also matches everything under
// it as a folder. With * as the only wildcard, taking each literal segment at
// its leftmost match is correct, so this is linear in the pattern and path.
func dsyPatternMatch(pattern, path string) bool {
	if !strings.Contains(pattern, "*") {
		folder := strings.TrimSuffix(pattern, "/")
		return path == folder || strings.HasPrefix(path, folder+"/")
	}
	parts := strings.Split(pattern, "*")
	first, last := parts[0], parts[len(parts)-1]
	if !strings.HasPrefix(path, first) {
		return false
	}
	rest := path[len(first):]
	for _, mid := range parts[1 : len(parts)-1] {
		i := strings.Index(rest, mid)
		if i < 0 {
			return false
		}
		rest = rest[i+len(mid):]
	}
	return len(rest) >= len(last) && strings.HasSuffix(rest, last)
}
