package server

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type glcInventoryParams struct {
	StartDate string `json:"StartDate,omitempty"`
	EndDate   string `json:"EndDate,omitempty"`
	Limit     string `json:"Limit,omitempty"`
	Marker    string `json:"Marker,omitempty"`
}

// glcJobParams is the InitiateJob body (the JobParameters payload).
type glcJobParams struct {
	Format                       string              `json:"Format"`
	Type                         string              `json:"Type"`
	ArchiveID                    string              `json:"ArchiveId"`
	Description                  string              `json:"Description"`
	SNSTopic                     string              `json:"SNSTopic"`
	RetrievalByteRange           string              `json:"RetrievalByteRange"`
	Tier                         string              `json:"Tier"`
	InventoryRetrievalParameters *glcInventoryParams `json:"InventoryRetrievalParameters"`
	SelectParameters             map[string]any      `json:"SelectParameters"`
	OutputLocation               map[string]any      `json:"OutputLocation"`
}

type glcJob struct {
	id, action, desc, tier, sns, format string
	archiveID, archiveDesc, archiveHash string
	archiveSize                         int64
	rangeStart, rangeEnd                int64 // inclusive, archive retrieval only
	created, completedAt                time.Time
	status                              string
	reads                               int
	data                                []byte // retrieved bytes or the inventory document
	held                                int64  // bytes of data counted in the store budget (inventories)
	gone                                bool   // the archive was deleted, so the output is gone too
	invParams                           *glcInventoryParams
	selectParams, outputLocation        map[string]any
	outputPath                          string
}

var glcActions = map[string]string{"archive-retrieval": "ArchiveRetrieval", "inventory-retrieval": "InventoryRetrieval", "select": "Select"}

// glcCheckJobParams validates what doesn't need the store.
func glcCheckJobParams(p *glcJobParams) *glcErr {
	if p.Type == "" {
		return glcMissing("Job type is required")
	}
	if _, ok := glcActions[p.Type]; !ok {
		return glcInvalid("Invalid job type %q. Types are archive-retrieval, inventory-retrieval and select", p.Type)
	}
	if err := glcCheckDescription("Job description", p.Description); err != nil {
		return err
	}
	switch p.Tier {
	case "", "Expedited", "Standard", "Bulk":
	default:
		return glcInvalid("Invalid tier %q. Tiers are Expedited, Standard and Bulk", p.Tier)
	}
	switch p.Type {
	case "archive-retrieval", "select":
		if p.ArchiveID == "" {
			return glcMissing("ArchiveId is required for %s jobs", p.Type)
		}
		if p.InventoryRetrievalParameters != nil || (p.Format != "" && p.Type == "archive-retrieval") {
			return glcInvalid("Format and InventoryRetrievalParameters apply only to inventory-retrieval jobs")
		}
	case "inventory-retrieval":
		if p.ArchiveID != "" || p.RetrievalByteRange != "" || p.SelectParameters != nil {
			return glcInvalid("ArchiveId, RetrievalByteRange and SelectParameters don't apply to inventory-retrieval jobs")
		}
		switch p.Format {
		case "", "JSON", "CSV":
		default:
			return glcInvalid("Invalid format %q. Formats are JSON and CSV", p.Format)
		}
		if ip := p.InventoryRetrievalParameters; ip != nil {
			for _, d := range []string{ip.StartDate, ip.EndDate} {
				if _, err := time.Parse(time.RFC3339, d); d != "" && err != nil {
					return glcInvalid("Invalid date %q. Use ISO 8601, for example 2024-01-02T15:04:05Z", d)
				}
			}
			if ip.Limit != "" {
				if n, err := strconv.ParseInt(ip.Limit, 10, 32); err != nil || n < 1 {
					return glcInvalid("Invalid inventory limit %q", ip.Limit)
				}
			}
		}
	}
	if p.Type == "select" {
		if p.SelectParameters == nil || p.OutputLocation == nil {
			return glcMissing("SelectParameters and OutputLocation are required for select jobs")
		}
		if t, _ := p.SelectParameters["ExpressionType"].(string); t != "SQL" {
			return glcInvalid("ExpressionType must be SQL")
		}
		if e, _ := p.SelectParameters["Expression"].(string); strings.TrimSpace(e) == "" {
			return glcMissing("Expression is required")
		}
		if glcSelectBucket(p) == "" {
			return glcMissing("OutputLocation.S3.BucketName is required")
		}
	} else if p.SelectParameters != nil || p.OutputLocation != nil {
		return glcInvalid("SelectParameters and OutputLocation apply only to select jobs")
	}
	return nil
}

func glcSelectBucket(p *glcJobParams) string {
	s3, _ := p.OutputLocation["S3"].(map[string]any)
	name, _ := s3["BucketName"].(string)
	return name
}

// glcParseRetrievalRange checks "start-end": start on a 1 MiB boundary, end
// one byte before a 1 MiB boundary or the archive's last byte.
func glcParseRetrievalRange(spec string, size int64) (int64, int64, *glcErr) {
	if spec == "" {
		return 0, size - 1, nil
	}
	a, b, ok := strings.Cut(spec, "-")
	start, err1 := strconv.ParseInt(a, 10, 64)
	end, err2 := strconv.ParseInt(b, 10, 64)
	if !ok || err1 != nil || err2 != nil || start < 0 || end < start || end >= size {
		return 0, 0, glcInvalid("Invalid RetrievalByteRange %q for an archive of %d bytes", spec, size)
	}
	if start%glcTreeChunk != 0 || ((end+1)%glcTreeChunk != 0 && end != size-1) {
		return 0, 0, glcInvalid("RetrievalByteRange %q must be megabyte aligned", spec)
	}
	return start, end, nil
}

func (st *glcStore) initiateJob(name string, p *glcJobParams) (*glcJob, *glcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return nil, err
	}
	j := &glcJob{id: glcRandomID(92), action: glcActions[p.Type], desc: p.Description, tier: p.Tier, sns: p.SNSTopic, created: glcNow(), status: "InProgress"}
	if j.tier == "" && p.Type != "inventory-retrieval" {
		j.tier = "Standard"
	}
	if j.sns == "" && v.notify != nil {
		event := map[string]string{"ArchiveRetrieval": "ArchiveRetrievalCompleted", "InventoryRetrieval": "InventoryRetrievalCompleted"}[j.action]
		for _, e := range v.notify.Events {
			if e == event {
				j.sns = v.notify.Topic
			}
		}
	}
	switch p.Type {
	case "archive-retrieval", "select":
		a, ok := v.archives[p.ArchiveID]
		if !ok {
			return nil, glcNotFound("Archive not found: %s", p.ArchiveID)
		}
		j.archiveID, j.archiveDesc, j.archiveHash, j.archiveSize = a.id, a.desc, a.hash, int64(len(a.data))
		if p.Type == "select" {
			j.selectParams, j.outputLocation = p.SelectParameters, p.OutputLocation
			s3, _ := p.OutputLocation["S3"].(map[string]any)
			prefix, _ := s3["Prefix"].(string)
			j.outputPath = strings.TrimSuffix(prefix, "/") + "/" + j.id
			break
		}
		start, end, rerr := glcParseRetrievalRange(p.RetrievalByteRange, j.archiveSize)
		if rerr != nil {
			return nil, rerr
		}
		if limit := st.bytesPerHourLocked(); limit > 0 && end-start+1 > limit {
			return nil, &glcErr{400, "PolicyEnforcedException", fmt.Sprintf("Retrieving %d bytes exceeds the data retrieval policy's limit of %d bytes per hour", end-start+1, limit)}
		}
		j.rangeStart, j.rangeEnd = start, end
		j.data = a.data[start : end+1] // archives are immutable, so sharing is safe
	case "inventory-retrieval":
		j.format = p.Format
		if j.format == "" {
			j.format = "JSON"
		}
		j.invParams = p.InventoryRetrievalParameters
	}
	if len(v.jobs) >= glcMaxJobs && !st.evictJobLocked(v) {
		return nil, glcLimit("A vault keeps at most %d jobs and none has completed", glcMaxJobs)
	}
	v.jobs[j.id] = j
	v.jobOrder = append(v.jobOrder, j.id)
	return j, nil
}

func (st *glcStore) bytesPerHourLocked() int64 {
	for _, r := range st.retrieval {
		if r.Strategy == "BytesPerHour" && r.BytesPerHour != nil {
			return *r.BytesPerHour
		}
	}
	return 0
}

func (st *glcStore) evictJobLocked(v *glcVault) bool {
	for i, id := range v.jobOrder {
		if j := v.jobs[id]; j.status != "InProgress" {
			st.bytes -= j.held
			delete(v.jobs, id)
			v.jobOrder = append(v.jobOrder[:i:i], v.jobOrder[i+1:]...)
			return true
		}
	}
	return false
}

// settleLocked advances a job one step per read: the first read after it
// starts still sees InProgress, the next sees it Succeeded.
// ponytail: job completion isn't published to SNS; the SNS store only
// returns message ids, so there'd be nothing to observe.
func (st *glcStore) settleLocked(v *glcVault, j *glcJob) {
	if j.status != "InProgress" {
		return
	}
	if j.reads == 0 {
		j.reads++
		return
	}
	j.status, j.completedAt = "Succeeded", glcNow()
	if j.action == "InventoryRetrieval" {
		doc := v.inventoryDocLocked(j)
		// Inventory documents are held until the job goes, so they share the
		// storage budget with archives.
		if st.reserveLocked(int64(len(doc))) != nil {
			j.status = "Failed"
			return
		}
		j.data, j.held = doc, int64(len(doc))
		st.bytes += j.held
		v.inventory = &glcInventory{date: j.completedAt}
		for _, a := range v.archives {
			v.inventory.count++
			v.inventory.size += int64(len(a.data))
		}
	}
}

// inventoryDocLocked renders the vault's archive list as JSON or CSV,
// honouring the job's date range, marker and limit.
func (v *glcVault) inventoryDocLocked(j *glcJob) []byte {
	archives := make([]*glcArchive, 0, len(v.archives))
	for _, a := range v.archives {
		archives = append(archives, a)
	}
	sort.Slice(archives, func(i, k int) bool {
		return archives[i].created.Before(archives[k].created) || (archives[i].created.Equal(archives[k].created) && archives[i].id < archives[k].id)
	})
	if ip := j.invParams; ip != nil {
		start, _ := time.Parse(time.RFC3339, ip.StartDate)
		end, _ := time.Parse(time.RFC3339, ip.EndDate)
		kept := archives[:0]
		skipping := ip.Marker != ""
		for _, a := range archives {
			if skipping {
				skipping = a.id != ip.Marker
				continue
			}
			if (ip.StartDate == "" || !a.created.Before(start)) && (ip.EndDate == "" || a.created.Before(end)) {
				kept = append(kept, a)
			}
		}
		archives = kept
		if n, err := strconv.ParseInt(ip.Limit, 10, 32); err == nil && int(n) < len(archives) {
			archives = archives[:n]
		}
	}
	if j.format == "CSV" {
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		_ = w.Write([]string{"ArchiveId", "ArchiveDescription", "CreationDate", "Size", "SHA256TreeHash"})
		for _, a := range archives {
			_ = w.Write([]string{a.id, a.desc, glcTime(a.created), strconv.Itoa(len(a.data)), a.hash})
		}
		w.Flush()
		return buf.Bytes()
	}
	list := make([]map[string]any, 0, len(archives))
	for _, a := range archives {
		list = append(list, map[string]any{"ArchiveId": a.id, "ArchiveDescription": a.desc, "CreationDate": glcTime(a.created), "Size": len(a.data), "SHA256TreeHash": a.hash})
	}
	raw, _ := json.Marshal(map[string]any{"VaultARN": glcVaultARN(v.name), "InventoryDate": glcTime(glcNow()), "ArchiveList": list})
	return raw
}

func (j *glcJob) describe(vault string) map[string]any {
	out := map[string]any{"JobId": j.id, "Action": j.action, "VaultARN": glcVaultARN(vault), "CreationDate": glcTime(j.created), "Completed": j.status != "InProgress", "StatusCode": j.status}
	if j.desc != "" {
		out["JobDescription"] = j.desc
	}
	if j.tier != "" {
		out["Tier"] = j.tier
	}
	if j.sns != "" {
		out["SNSTopic"] = j.sns
	}
	switch j.status {
	case "Succeeded":
		out["CompletionDate"] = glcTime(j.completedAt)
		out["StatusMessage"] = "Succeeded"
	case "Failed":
		out["CompletionDate"] = glcTime(j.completedAt)
		out["StatusMessage"] = "The inventory exceeds Stackyard's Glacier storage budget"
	}
	switch j.action {
	case "ArchiveRetrieval":
		out["ArchiveId"], out["ArchiveSizeInBytes"], out["ArchiveSHA256TreeHash"] = j.archiveID, j.archiveSize, j.archiveHash
		out["RetrievalByteRange"] = fmt.Sprintf("%d-%d", j.rangeStart, j.rangeEnd)
		if !j.gone {
			out["SHA256TreeHash"] = glcTreeHash(j.data)
		}
	case "InventoryRetrieval":
		ip := glcInventoryParams{}
		if j.invParams != nil {
			ip = *j.invParams
		}
		out["InventoryRetrievalParameters"] = map[string]any{"Format": j.format, "StartDate": ip.StartDate, "EndDate": ip.EndDate, "Limit": ip.Limit, "Marker": ip.Marker}
		if j.status == "Succeeded" {
			out["InventorySizeInBytes"] = int64(len(j.data))
		}
	case "Select":
		out["ArchiveId"], out["ArchiveSizeInBytes"], out["ArchiveSHA256TreeHash"] = j.archiveID, j.archiveSize, j.archiveHash
		out["SelectParameters"], out["OutputLocation"], out["JobOutputPath"] = j.selectParams, j.outputLocation, j.outputPath
	}
	return out
}

func (st *glcStore) jobLocked(name, id string) (*glcVault, *glcJob, *glcErr) {
	v, err := st.vaultLocked(name)
	if err != nil {
		return nil, nil, err
	}
	j, ok := v.jobs[id]
	if !ok {
		return nil, nil, glcNotFound("Job not found: %s", id)
	}
	return v, j, nil
}

func (st *glcStore) describeJob(name, id string) (map[string]any, *glcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, j, err := st.jobLocked(name, id)
	if err != nil {
		return nil, err
	}
	st.settleLocked(v, j)
	return j.describe(name), nil
}

func (st *glcStore) listJobs(name, status, completed, marker string, limit int) (map[string]any, *glcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return nil, err
	}
	jobs := make([]*glcJob, 0, len(v.jobOrder))
	for _, id := range v.jobOrder {
		j := v.jobs[id]
		st.settleLocked(v, j)
		if (status == "" || j.status == status) && (completed == "" || completed == strconv.FormatBool(j.status != "InProgress")) {
			jobs = append(jobs, j)
		}
	}
	page, next, perr := glcPage(jobs, func(j *glcJob) string { return j.id }, marker, limit)
	if perr != nil {
		return nil, perr
	}
	list := make([]map[string]any, 0, len(page))
	for _, j := range page {
		list = append(list, j.describe(name))
	}
	out := map[string]any{"JobList": list}
	if next != "" {
		out["Marker"] = next
	}
	return out, nil
}

// glcJobOutput is what GetJobOutput sends, copied out under the store lock.
type glcJobOutput struct {
	kind, desc string // kind: archive, json or csv
	data       []byte
}

func (st *glcStore) jobOutput(name, id string) (*glcJobOutput, *glcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, j, err := st.jobLocked(name, id)
	if err != nil {
		return nil, err
	}
	st.settleLocked(v, j)
	if j.status != "Succeeded" {
		return nil, glcInvalid("The job is not currently available for download: %s", id)
	}
	if j.gone {
		return nil, glcNotFound("The archive for job %s has been deleted", id)
	}
	switch j.action {
	case "ArchiveRetrieval":
		return &glcJobOutput{kind: "archive", desc: j.archiveDesc, data: j.data}, nil
	case "InventoryRetrieval":
		return &glcJobOutput{kind: strings.ToLower(j.format), data: j.data}, nil
	}
	return nil, glcInvalid("Select job output is written to the job's S3 output location, not returned by GetJobOutput")
}
